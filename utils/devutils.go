package utils

import "os"

func DisableHttpProxy() {
	os.Setenv("http_proxy", "")
	os.Setenv("https_proxy", "")
}
