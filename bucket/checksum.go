package bucket

import "fmt"

func getChecksumPath(key, versionID string) string {
	return fmt.Sprintf("_pickle/checksum/%s_%s.sha256", key, versionID)
}
