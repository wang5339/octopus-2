package responsebody

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/bestruirui/octopus/internal/conf"
)

const DefaultMaxBytes int64 = 64 * 1024 * 1024

// MaxBytes 是非流式上游成功响应体的读取上限。
// 图像生成的 b64_json 合法响应可能明显大于普通文本响应，因此默认值高于
// request body 和 SSE 单事件限制；如需更大图像响应，可通过
// OCTOPUS_RELAY_MAX_RESPONSE_BODY_SIZE 覆盖。
var MaxBytes = DefaultMaxBytes

func init() {
	envName := strings.ToUpper(conf.APP_NAME) + "_RELAY_MAX_RESPONSE_BODY_SIZE"
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return
	}
	if value, err := strconv.ParseInt(raw, 10, 64); err == nil && value > 0 {
		MaxBytes = value
	}
}

func ReadAll(reader io.Reader) ([]byte, error) {
	if reader == nil {
		return nil, nil
	}

	body, err := io.ReadAll(io.LimitReader(reader, MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > MaxBytes {
		return nil, fmt.Errorf("upstream response body exceeds %d byte limit", MaxBytes)
	}
	return body, nil
}
