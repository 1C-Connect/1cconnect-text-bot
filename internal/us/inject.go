package us

import (
	"github.com/gin-gonic/gin"
	"github.com/hooklift/gowsdl/soap"
)

type UsServer struct {
	Addr string `yaml:"addr"`
}

func Inject(us UsServer, login, password string) gin.HandlerFunc {
	return func(c *gin.Context) {
		soapcl := soap.NewClient(
			us.Addr,
			soap.WithBasicAuth(login, password),
		)

		c.Set("soapcl", soapcl)
	}
}

func InjectMTOM(us UsServer, login, password string) gin.HandlerFunc {
	return func(c *gin.Context) {
		soapcl := soap.NewClient(
			us.Addr,
			soap.WithBasicAuth(login, password),
			soap.WithMTOM(), // WithMIMEMultipartAttachments(),
		)

		c.Set("soapclmtom", soapcl)
	}
}
