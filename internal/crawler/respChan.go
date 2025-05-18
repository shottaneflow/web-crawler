package crawler

type RespChan struct {
	htmlContent string
	links       []string
	err         error
}
