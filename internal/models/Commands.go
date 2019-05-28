package models

//TODO: add the right copyright header

// Json request from EdgeX to RSP gateway
type JsonRequest struct {
	JsonRpc string `json:"jsonrpc"`
	Id      string `json:"id"`
	Method  string `json:"method"`
}

// Json response from the RSP gateway
type JsonResponse struct {
	JsonRpc string      `json:"jsonrpc"`
	Id      string      `json:"id"`
	Result  interface{} `json:"result"`
	Error   interface{} `json:"error"`
}
