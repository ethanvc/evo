package httpproto

type ResponseDto struct {
	Code *int   `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func (r *ResponseDto) SetCode(code int) {
	r.Code = &code
}

func (r *ResponseDto) SetMsg(msg string) {
	r.Msg = msg
}

func (r *ResponseDto) SetData(data any) {
	r.Data = data
}

func (r *ResponseDto) GetCode() int {
	if r.Code == nil {
		return -1
	}
	return *r.Code
}

func (r *ResponseDto) GetMsg() string {
	return r.Msg
}

func (r *ResponseDto) GetData() any {
	return r.Data
}
