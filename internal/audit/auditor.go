package audit

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type Auditor struct {
	w io.Writer
}

func NewAuditor(w io.Writer) *Auditor {
	return &Auditor{w: w}
}

func (a *Auditor) Log(clientID string, scopes []string, method, path string) {
	fmt.Fprintf(a.w, "[%s] client=%s scopes=[%s] %s %s\n",
		time.Now().UTC().Format(time.RFC3339), clientID, strings.Join(scopes, ","), method, path)
}
