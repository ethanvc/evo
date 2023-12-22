package plog

/*
For RPC or http request, report `evo_server_event_total` when execution finished. The report event format is:
`REQ:${code}:${event}`, for example: `REQ:NOT_FOUND:UserNotFound`
*/
