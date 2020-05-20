package bouncer

import (
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

type Method string

const (
	Get     = "GET"
	Post    = "POST"
	Options = "OPTIONS"
)

type ApiHandler func(map[string]string, events.APIGatewayProxyRequest) *events.APIGatewayProxyResponse

type handlerNode struct {
	handler       *ApiHandler
	subnodes      map[string]handlerNode
	parameterName string
}

type Bouncer struct {
	BasePath     string
	getHandlers  handlerNode
	postHandlers handlerNode
}

func New(basePath string) *Bouncer {
	return &Bouncer{
		BasePath: basePath,
		getHandlers: handlerNode{
			subnodes: make(map[string]handlerNode),
		},
		postHandlers: handlerNode{
			subnodes: make(map[string]handlerNode),
		},
	}
}

func (h *handlerNode) update(path []string, handler ApiHandler) {
	if len(path) == 0 {
		h.handler = &handler
		return
	}

	thisPart := path[0]
	// if it's a pattern, store the name then continue
	if strings.HasPrefix(thisPart, "{") && strings.HasSuffix(thisPart, "}") {
		// FIXME: should be a better way to do this
		h.parameterName = strings.Replace(strings.Replace(thisPart, "{", "", -1), "}", "", -1)
		h.update(path[1:], handler)
		return
	}

	if subnode, ok := h.subnodes[thisPart]; ok {
		subnode.update(path[1:], handler)
	} else {
		newSubnode := handlerNode{
			subnodes: make(map[string]handlerNode),
		}
		newSubnode.update(path[1:], handler)
		h.subnodes[thisPart] = newSubnode
	}
}

func (h *handlerNode) get(path string) (*ApiHandler, map[string]string) {
	pathElements := strings.Split(path, "/")
	parameters := make(map[string]string)
	currNode := h
	for currNode != nil && len(pathElements) > 0 {
		if nextNode, ok := currNode.subnodes[pathElements[0]]; ok {
			currNode = &nextNode
		} else if currNode.parameterName != "" {
			parameters[currNode.parameterName] = pathElements[0]
		}
		pathElements = pathElements[1:]
	}

	if len(pathElements) == 0 {
		return currNode.handler, parameters
	}
	return nil, nil
}

func (b *Bouncer) Handle(method Method, pattern string, handler ApiHandler) {
	// FIXME: for now just prepend the path with the base path
	path := strings.Split(b.BasePath+pattern, "/")
	switch method {
	case Get:
		b.getHandlers.update(path, handler)
	case Post:
		b.postHandlers.update(path, handler)
	default:
		b.getHandlers.update(path, handler)
	}
}

func (b *Bouncer) Route(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	var handler *ApiHandler
	var parameters map[string]string
	switch req.HTTPMethod {
	case Get:
		handler, parameters = b.getHandlers.get(req.Path)
	case Post:
		handler, parameters = b.postHandlers.get(req.Path)
	default:
		handler, parameters = b.getHandlers.get(req.Path)
	}
	if handler == nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 404,
			Body:       fmt.Sprintf("No resource at %v %v", req.HTTPMethod, req.Path),
		}, nil
	}

	return (*handler)(parameters, req), nil
}
