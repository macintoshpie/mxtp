package bouncer

import (
	"errors"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

type apiHandler func(map[string]string, events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error)

type handlerNode struct {
	handler       *apiHandler
	subnodes      map[string]handlerNode
	parameterName string
}

type Bouncer struct {
	BasePath string
	handlers handlerNode
}

func New(basePath string) *Bouncer {
	return &Bouncer{
		BasePath: basePath,
		handlers: handlerNode{
			subnodes: make(map[string]handlerNode),
		},
	}
}

func (h *handlerNode) update(path []string, handler apiHandler) {
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

func (h *handlerNode) get(path string) (*apiHandler, map[string]string) {
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

func (b *Bouncer) Handle(pattern string, handler apiHandler) {
	// FIXME: for now just prepend the path with the base path
	path := strings.Split(b.BasePath+pattern, "/")
	b.handlers.update(path, handler)
}

func (b *Bouncer) Route(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	handler, parameters := b.handlers.get(req.Path)
	if handler == nil {
		return nil, errors.New("No handler found")
	}
	return (*handler)(parameters, req)
}
