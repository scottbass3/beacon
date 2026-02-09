package registry

type RequestLog struct {
	Method  string
	URL     string
	Headers map[string][]string
	Status  int
}

type RequestLogger func(RequestLog)
