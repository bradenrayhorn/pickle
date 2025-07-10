package bucket

import (
	"encoding/hex"
	"fmt"
	"strings"
)

func getChecksumPath(key string) string {
	return fmt.Sprintf("_pickle/checksum/%s.sha256", hex.EncodeToString([]byte(key)))
}

func isDataFile(key string) bool {
	parts := strings.Split(key, ".")

	return len(parts) > 2 && parts[len(parts)-2] == "age" && !strings.HasPrefix(key, "_pickle/")
}

func isChecksumFile(key string) bool {
	return strings.HasPrefix(key, "_pickle/checksum")
}
