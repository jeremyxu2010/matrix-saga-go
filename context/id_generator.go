package context

import "github.com/satori/go.uuid"

type IdGenerator struct{
}

func (g *IdGenerator)NextId()string{
	return uuid.NewV4().String()
}

var idGenerator *IdGenerator

func init()  {
	idGenerator = &IdGenerator{}
}

