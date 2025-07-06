package bucket

import (
	"fmt"
	"strings"
)

func getChecksumPath(key, versionID string) string {
	return fmt.Sprintf("_pickle/checksum/%s_%s.sha256", key, versionID)
}

func isDataFile(key string) bool {
	return strings.HasSuffix(key, ".age") && !strings.HasPrefix(key, "_pickle/")
}

func isChecksumFile(key string) bool {
	return strings.HasPrefix(key, "_pickle/checksum")
}
