package relay

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/helper"
	dbmodel "github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/relay/balancer"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/transformer/inbound"
	"github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/bestruirui/octopus/internal/transformer/outbound"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/gin-gonic/gin"
	"github.com/tmaxmax/go-sse"
)

const (
	nonStreamRequestTimeout     = 120 * time.Second
	maxUpstreamErrorBodyBytes   = 16 * 1024
	upstreamErrorBodyTruncation = "\n... [truncated]"
)

// Handler 处理入站请求并转发到上游服务
func Handler(inboundType inbound.InboundType, c *gin.Context) {
	// 解析请求
	internalRequest, inAdapter, err := parseRequest(inboundType, c)
	if err != nil {
		return
	}
	supportedModels := c.GetString("supported_models")
	if supportedModels != "" {
		supportedModelsArray := strings.Split(supportedModels, ",")
		if !slices.Contains(supportedModelsArray, internalRequest.Model) {
			resp.Error(c, http.StatusBadRequest, "model not allowed for this API key")
			return
		}
	}

	requestModel := internalRequest.Model
	apiKeyID := c.GetInt("api_key_id")

	// 获取通道分组
	group, err := op.GroupGetEnabledMap(requestModel, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusNotFound, "model not found")
		return
	}

	// 创建迭代器（策略排序 + 粘性优先）
	iter := balancer.NewIterator(group, apiKeyID, requestModel)
	if iter.Len() == 0 {
		resp.Error(c, http.StatusServiceUnavailable, "no available channel")
		return
	}

	// 初始化 Metrics
	metrics := NewRelayMetrics(apiKeyID, requestModel, internalRequest)

	// 请求级上下文
	req := &relayRequest{
		c:               c,
		inAdapter:       inAdapter,
		internalRequest: internalRequest,
		metrics:         metrics,
		apiKeyID:        apiKeyID,
		requestModel:    requestModel,
		iter:            iter,
	}

	var lastErr error

	for iter.Next() {
		select {
		case <-c.Request.Context().Done():
			log.Infof("request context canceled, stopping retry")
			metrics.Save(c.Request.Context(), false, context.Canceled, iter.Attempts())
			return
		default:
		}

		item := iter.Item()

		// 获取通道
		channel, err := op.ChannelGet(item.ChannelID, c.Request.Context())
		if err != nil {
			log.Warnf("failed to get channel %d: %v", item.ChannelID, err)
			iter.Skip(item.ChannelID, 0, fmt.Sprintf("channel_%d", item.ChannelID), fmt.Sprintf("channel not found: %v", err))
			lastErr = err
			continue
		}
		if !channel.Enabled {
			iter.Skip(channel.ID, 0, channel.Name, "channel disabled")
			continue
		}

		usedKey := channel.GetChannelKey()
		if usedKey.ChannelKey == "" {
			iter.Skip(channel.ID, 0, channel.Name, "no available key")
			continue
		}

		// 熔断检查
		if iter.SkipCircuitBreak(channel.ID, usedKey.ID, channel.Name) {
			continue
		}

		upstreamModelName := item.ModelName
		effectiveType := resolveModelOutboundType(*channel, upstreamModelName)

		// 出站适配器：模型级协议覆盖优先，其次保留 Zen 渠道按模型动态选择协议。
		outAdapter := outbound.GetForModel(effectiveType, upstreamModelName)
		if outAdapter == nil {
			iter.Skip(channel.ID, usedKey.ID, channel.Name, fmt.Sprintf("unsupported channel type: %d", effectiveType))
			continue
		}

		// 类型兼容性检查使用最终生效的出站协议。
		if internalRequest.IsEmbeddingRequest() && !outbound.IsEmbeddingChannelType(effectiveType) {
			iter.Skip(channel.ID, usedKey.ID, channel.Name, "channel type not compatible with embedding request")
			continue
		}
		if internalRequest.IsImageGenerationRequest() && !outbound.IsImageGenerationChannelType(effectiveType) {
			iter.Skip(channel.ID, usedKey.ID, channel.Name, "channel type not compatible with image generation request")
			continue
		}
		if internalRequest.IsChatRequest() && !outbound.IsChatChannelType(effectiveType) {
			iter.Skip(channel.ID, usedKey.ID, channel.Name, "channel type not compatible with chat request")
			continue
		}

		// 设置实际模型
		internalRequest.Model = upstreamModelName

		log.Infof("request model %s, mode: %d, forwarding to channel: %s model: %s outbound_type: %d (attempt %d/%d, sticky=%t)",
			requestModel, group.Mode, channel.Name, upstreamModelName, effectiveType,
			iter.Index()+1, iter.Len(), iter.IsSticky())

		// 构造尝试级上下文 -- 只写变化的 4 个字段
		ra := &relayAttempt{
			relayRequest:         req,
			outAdapter:           outAdapter,
			channel:              channel,
			usedKey:              usedKey,
			firstTokenTimeOutSec: group.FirstTokenTimeOut,
		}

		result := ra.attempt()
		if result.Success {
			metrics.Save(c.Request.Context(), true, nil, iter.Attempts())
			return
		}
		if result.Written {
			metrics.Save(c.Request.Context(), false, result.Err, iter.Attempts())
			return
		}

		// 429（速率限制）: 尝试同渠道的下一个可用 key，而不是直接跳到下一个渠道
		if result.StatusCode == http.StatusTooManyRequests {
			const maxRetryKeysPerChannel = 5 // 最大重试次数，防止无限循环
			retryCount := 0
			for retryCount < maxRetryKeysPerChannel {
				retryCount++
				// 把当前 key 的最新状态（StatusCode=429, LastUseTimeStamp）同步到本地
				// channel.Keys 快照，确保 GetChannelKey 能正确跳过它，避免死循环
				for i := range channel.Keys {
					if channel.Keys[i].ID == ra.usedKey.ID {
						channel.Keys[i].StatusCode = ra.usedKey.StatusCode
						channel.Keys[i].LastUseTimeStamp = ra.usedKey.LastUseTimeStamp
						break
					}
				}
				nextKey := channel.GetChannelKey()
				if nextKey.ChannelKey == "" || nextKey.ID == ra.usedKey.ID {
					log.Infof("no more available keys for channel %s after %d retries", channel.Name, retryCount)
					break // 同渠道无更多可用 key，移到下一个渠道
				}
				log.Infof("key %d got 429, retrying channel %s with key %d (retry %d/%d)", ra.usedKey.ID, channel.Name, nextKey.ID, retryCount, maxRetryKeysPerChannel)
				ra.usedKey = nextKey
				retryResult := ra.attempt()
				if retryResult.Success {
					metrics.Save(c.Request.Context(), true, nil, iter.Attempts())
					return
				}
				if retryResult.Written {
					metrics.Save(c.Request.Context(), false, retryResult.Err, iter.Attempts())
					return
				}
				if retryResult.StatusCode != http.StatusTooManyRequests {
					lastErr = retryResult.Err
					break // 非 429 错误，移到下一个渠道
				}
				// 继续尝试下一个 key
			}
			if retryCount >= maxRetryKeysPerChannel {
				log.Warnf("reached max retry limit (%d) for channel %s, moving to next channel", maxRetryKeysPerChannel, channel.Name)
			}
		}

		lastErr = result.Err
	}

	// 所有通道都失败
	metrics.Save(c.Request.Context(), false, lastErr, iter.Attempts())
	resp.Error(c, http.StatusBadGateway, "all channels failed")
}

// attempt 统一管理一次通道尝试的完整生命周期
func (ra *relayAttempt) attempt() attemptResult {
	span := ra.iter.StartAttempt(ra.channel.ID, ra.usedKey.ID, ra.channel.Name)

	// 转发请求
	statusCode, fwdErr := ra.forward()

	// 更新 channel key 状态
	ra.usedKey.StatusCode = statusCode
	ra.usedKey.LastUseTimeStamp = time.Now().Unix()

	if fwdErr == nil {
		// ====== 成功 ======
		ra.collectResponse()
		ra.usedKey.TotalCost += ra.metrics.Stats.InputCost + ra.metrics.Stats.OutputCost
		op.ChannelKeyUpdate(ra.usedKey)

		span.End(dbmodel.AttemptSuccess, statusCode, "")

		// 熔断器：记录成功
		balancer.RecordSuccess(ra.channel.ID, ra.usedKey.ID, ra.internalRequest.Model)
		// 会话保持：更新粘性记录
		balancer.SetSticky(ra.apiKeyID, ra.requestModel, ra.channel.ID, ra.usedKey.ID)

		return attemptResult{Success: true}
	}

	// ====== 失败 ======
	op.ChannelKeyUpdate(ra.usedKey)
	span.End(dbmodel.AttemptFailed, statusCode, fwdErr.Error())

	// 熔断器：记录失败（context canceled / 客户端断开不计入，避免误触发熔断）
	if !errors.Is(fwdErr, context.Canceled) && !errors.Is(fwdErr, context.DeadlineExceeded) {
		balancer.RecordFailure(ra.channel.ID, ra.usedKey.ID, ra.internalRequest.Model)
	} else {
		log.Infof("skipping circuit breaker failure for channel %s: client context canceled", ra.channel.Name)
	}

	written := ra.c.Writer.Written()
	if written {
		ra.collectResponse()
	}
	return attemptResult{
		Success:    false,
		Written:    written,
		Err:        fmt.Errorf("channel %s failed: %v", ra.channel.Name, fwdErr),
		StatusCode: statusCode,
	}
}

func resolveModelOutboundType(channel dbmodel.Channel, modelName string) outbound.OutboundType {
	trimmedModel := strings.TrimSpace(modelName)
	for _, item := range channel.ModelProtocolOverrides {
		if strings.EqualFold(strings.TrimSpace(item.Model), trimmedModel) {
			return item.Type
		}
	}
	return channel.Type
}

// parseRequest 解析并验证入站请求
func parseRequest(inboundType inbound.InboundType, c *gin.Context) (*model.InternalLLMRequest, model.Inbound, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return nil, nil, err
	}

	inAdapter := inbound.Get(inboundType)
	internalRequest, err := inAdapter.TransformRequest(c.Request.Context(), body)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return nil, nil, err
	}

	// Pass through the original query parameters
	internalRequest.Query = c.Request.URL.Query()

	if err := internalRequest.Validate(); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return nil, nil, err
	}

	return internalRequest, inAdapter, nil
}

// forward 转发请求到上游服务
func (ra *relayAttempt) forward() (int, error) {
	ctx := ra.c.Request.Context()

	// 构建出站请求
	requestForAttempt, err := helper.CloneInternalLLMRequest(ra.internalRequest)
	if err != nil {
		log.Warnf("failed to clone request: %v", err)
		return 0, fmt.Errorf("failed to clone request: %w", err)
	}
	if err := helper.ApplyParamOverride(requestForAttempt, ra.channel.ParamOverride); err != nil {
		log.Warnf("failed to apply param override: %v", err)
		return 0, fmt.Errorf("failed to apply param override: %w", err)
	}
	outboundRequest, err := ra.outAdapter.TransformRequest(
		ctx,
		requestForAttempt,
		ra.channel.GetBaseUrl(),
		ra.usedKey.ChannelKey,
	)
	if err != nil {
		log.Warnf("failed to create request: %v", err)
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	// 复制请求头
	ra.copyHeaders(outboundRequest)

	// 发送请求
	response, err := ra.sendRequest(outboundRequest)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer response.Body.Close()

	// 检查响应状态
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, err := readUpstreamErrorBody(response.Body)
		if err != nil {
			return response.StatusCode, fmt.Errorf("failed to read response body: %w", err)
		}
		return response.StatusCode, fmt.Errorf("upstream error: %d: %s", response.StatusCode, body)
	}

	// 处理响应
	if ra.internalRequest.Stream != nil && *ra.internalRequest.Stream {
		if err := ra.handleStreamResponse(ctx, response); err != nil {
			return 0, err
		}
		return response.StatusCode, nil
	}
	if err := ra.handleResponse(ctx, response); err != nil {
		return 0, err
	}
	return response.StatusCode, nil
}

func readUpstreamErrorBody(body io.Reader) (string, error) {
	if body == nil {
		return "", nil
	}
	data, err := io.ReadAll(io.LimitReader(body, maxUpstreamErrorBodyBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxUpstreamErrorBodyBytes {
		data = data[:maxUpstreamErrorBodyBytes]
		return string(data) + upstreamErrorBodyTruncation, nil
	}
	return string(data), nil
}

// copyHeaders 复制请求头，过滤 hop-by-hop 头
func (ra *relayAttempt) copyHeaders(outboundRequest *http.Request) {
	for key, values := range ra.c.Request.Header {
		if hopByHopHeaders[strings.ToLower(key)] {
			continue
		}
		// 保留多值 header（如 Cookie、Accept）
		outboundRequest.Header.Del(key)
		for _, value := range values {
			outboundRequest.Header.Add(key, value)
		}
	}
	if len(ra.channel.CustomHeader) > 0 {
		for _, header := range ra.channel.CustomHeader {
			outboundRequest.Header.Set(header.HeaderKey, header.HeaderValue)
		}
	}
}

// sendRequest 发送 HTTP 请求
func (ra *relayAttempt) sendRequest(req *http.Request) (*http.Response, error) {
	httpClient, err := helper.ChannelHttpClient(ra.channel)
	if err != nil {
		log.Warnf("failed to get http client: %v", err)
		return nil, err
	}

	var cancel context.CancelFunc
	if ra.internalRequest.Stream == nil || !*ra.internalRequest.Stream {
		ctx, c := context.WithTimeout(req.Context(), nonStreamRequestTimeout)
		cancel = c
		req = req.WithContext(ctx)
	}

	response, err := httpClient.Do(req)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		log.Warnf("failed to send request: %v", err)
		return nil, err
	}
	if cancel != nil {
		response.Body = &cancelOnCloseReadCloser{ReadCloser: response.Body, cancel: cancel}
	}

	return response, nil
}

type cancelOnCloseReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (r *cancelOnCloseReadCloser) Close() error {
	err := r.ReadCloser.Close()
	r.cancel()
	return err
}

// handleStreamResponse 处理流式响应
func (ra *relayAttempt) handleStreamResponse(ctx context.Context, response *http.Response) error {
	if ct := response.Header.Get("Content-Type"); ct != "" && !strings.Contains(strings.ToLower(ct), "text/event-stream") {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 16*1024))
		return fmt.Errorf("upstream returned non-SSE content-type %q for stream request: %s", ct, string(body))
	}

	// 设置 SSE 响应头
	ra.c.Header("Content-Type", "text/event-stream")
	ra.c.Header("Cache-Control", "no-cache")
	ra.c.Header("Connection", "keep-alive")
	ra.c.Header("X-Accel-Buffering", "no")

	firstToken := true

	type sseReadResult struct {
		data string
		err  error
	}
	results := make(chan sseReadResult, 1)

	// 确保在函数退出时关闭 response.Body，触发 goroutine 退出
	defer func() {
		_ = response.Body.Close()
	}()

	go func() {
		defer close(results)
		readCfg := &sse.ReadConfig{MaxEventSize: maxSSEEventSize}
		for ev, err := range sse.Read(response.Body, readCfg) {
			if err != nil {
				select {
				case results <- sseReadResult{err: err}:
				case <-ctx.Done():
					// 主 goroutine 已退出，停止发送
					return
				}
				return
			}
			select {
			case results <- sseReadResult{data: ev.Data}:
			case <-ctx.Done():
				// 主 goroutine 已退出，停止发送
				return
			}
		}
	}()

	var firstTokenTimer *time.Timer
	var firstTokenC <-chan time.Time
	if firstToken && ra.firstTokenTimeOutSec > 0 {
		firstTokenTimer = time.NewTimer(time.Duration(ra.firstTokenTimeOutSec) * time.Second)
		firstTokenC = firstTokenTimer.C
		defer func() {
			if firstTokenTimer != nil {
				firstTokenTimer.Stop()
			}
		}()
	}

	for {
		select {
		case <-ctx.Done():
			log.Infof("client disconnected, stopping stream")
			return nil
		case <-firstTokenC:
			log.Warnf("first token timeout (%ds), switching channel", ra.firstTokenTimeOutSec)
			return fmt.Errorf("first token timeout (%ds)", ra.firstTokenTimeOutSec)
		case r, ok := <-results:
			if !ok {
				log.Infof("stream end")
				return nil
			}
			if r.err != nil {
				log.Warnf("failed to read event: %v", r.err)
				return fmt.Errorf("failed to read stream event: %w", r.err)
			}

			data, err := ra.transformStreamData(ctx, r.data)
			if err != nil || len(data) == 0 {
				continue
			}
			if firstToken {
				ra.metrics.SetFirstTokenTime(time.Now())
				firstToken = false
				if firstTokenTimer != nil {
					if !firstTokenTimer.Stop() {
						select {
						case <-firstTokenTimer.C:
						default:
						}
					}
					firstTokenTimer = nil
					firstTokenC = nil
				}
			}

			ra.c.Writer.Write(data)
			ra.c.Writer.Flush()
		}
	}
}

// transformStreamData 转换流式数据
func (ra *relayAttempt) transformStreamData(ctx context.Context, data string) ([]byte, error) {
	internalStream, err := ra.outAdapter.TransformStream(ctx, []byte(data))
	if err != nil {
		log.Warnf("failed to transform stream: %v", err)
		return nil, err
	}
	if internalStream == nil {
		return nil, nil
	}

	inStream, err := ra.inAdapter.TransformStream(ctx, internalStream)
	if err != nil {
		log.Warnf("failed to transform stream: %v", err)
		return nil, err
	}

	return inStream, nil
}

// handleResponse 处理非流式响应
func (ra *relayAttempt) handleResponse(ctx context.Context, response *http.Response) error {
	internalResponse, err := ra.outAdapter.TransformResponse(ctx, response)
	if err != nil {
		log.Warnf("failed to transform response: %v", err)
		return fmt.Errorf("failed to transform outbound response: %w", err)
	}

	inResponse, err := ra.inAdapter.TransformResponse(ctx, internalResponse)
	if err != nil {
		log.Warnf("failed to transform response: %v", err)
		return fmt.Errorf("failed to transform inbound response: %w", err)
	}

	ra.c.Data(http.StatusOK, "application/json", inResponse)
	return nil
}

// collectResponse 收集响应信息
func (ra *relayAttempt) collectResponse() {
	internalResponse, err := ra.inAdapter.GetInternalResponse(ra.c.Request.Context())
	if err != nil || internalResponse == nil {
		return
	}

	ra.metrics.SetInternalResponse(internalResponse, ra.internalRequest.Model)
}
