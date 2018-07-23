package proxy

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"
)

func getFrontTemplate(s Service) string {
	tmplString := `
{{- range $sd := .ServiceDest}}
    {{- if eq .ReqMode "http"}}
        {{- if ne $.CompressionAlgo ""}}
    compression algo {{$.CompressionAlgo}}
            {{- if ne $.CompressionType ""}}
    compression type {{$.CompressionType}}
            {{- end}}
        {{- end}}
        {{- if ne $sd.Port ""}}
    acl url_{{$.AclName}}{{$sd.Port}}_{{.Index}}{{range .ServicePath}} {{if eq $.PathType ""}}path_beg{{end}}{{if ne $.PathType ""}}{{$.PathType}}{{end}} {{.}}{{end}}
            {{- if .ServicePathExclude}}
    acl url_exclude_{{$.AclName}}{{$sd.Port}}_{{.Index}}{{range .ServicePathExclude}} {{if eq $.PathType ""}}path_beg{{end}}{{if ne $.PathType ""}}{{$.PathType}}{{end}} {{.}}{{end}}
            {{- end}}
            {{- if $sd.ServiceDomain}}
    acl domain_{{$.AclName}}{{$sd.Port}}_{{$sd.Index}} {{$.ServiceDomainAlgo}} -i{{range $sd.ServiceDomain}} {{.}}{{end}}
            {{- end}}
            {{- if $sd.ServiceHeader}}{{$skIndex := 0}}
                {{- range $key, $value := $sd.ServiceHeader}}
    acl hdr_{{$.AclName}}{{$sd.Port}}_{{incIndex}} hdr({{$key}}) {{$value}}
                {{- end}}
            {{- end}}
        {{- end}}
        {{- if gt $sd.HttpsPort 0}}
    acl url_https_{{$.AclName}}{{$sd.HttpsPort}}_{{.Index}}{{range .ServicePath}} {{if eq $.PathType ""}}path_beg{{end}}{{if ne $.PathType ""}}{{$.PathType}}{{end}} {{.}}{{end}}
            {{- if .ServicePathExclude}}
    acl url_exclude_https_{{$.AclName}}{{$sd.HttpsPort}}_{{.Index}}{{range .ServicePathExclude}} {{if eq $.PathType ""}}path_beg{{end}}{{if ne $.PathType ""}}{{$.PathType}}{{end}} {{.}}{{end}}
            {{- end}}
            {{- if $sd.ServiceDomain}}
    acl domain_https_{{$.AclName}}{{$sd.HttpsPort}}_{{$sd.Index}} {{$.ServiceDomainAlgo}} -i{{range $sd.ServiceDomain}} {{.}}{{end}}
            {{- end}}
        {{- end}}
        {{- if $sd.IncludeSrcPortACL }}
    {{$sd.SrcPortAcl}}
        {{- end }}
        {{- if $sd.IncludeSrcHttpsPortACL }}
    {{$sd.SrcHttpsPortAcl}}
        {{- end }}
        {{- $length := len .UserAgent.Value}}{{if gt $length 0}}
    acl user_agent_{{$.AclName}}_{{.UserAgent.AclName}}_{{.Index}} hdr_sub(User-Agent) -i{{range .UserAgent.Value}} {{.}}{{end}}
        {{- end}}
    {{- end}}
    {{- range $rd := $sd.RedirectFromDomain}}
    http-request redirect code 301 prefix http://{{index $sd.ServiceDomain 0}} if { hdr_beg(host) -i {{$rd}} }
    {{- end}}
{{- end}}
{{- if $.RedirectWhenHttpProto}}
    {{- range .ServiceDest}}
        {{- if eq .ReqMode "http"}}
           {{- if ne .Port ""}}
    acl is_{{$.AclName}}_http hdr(X-Forwarded-Proto) http
    http-request redirect scheme https{{if .HttpsRedirectCode}} code {{.HttpsRedirectCode}}{{end}} if is_{{$.AclName}}_http url_{{$.AclName}}{{.Port}}_{{.Index}}{{if .ServiceDomain}} domain_{{$.AclName}}{{.Port}}_{{.Index}}{{end}}{{.SrcPortAclName}}
            {{- end}}
        {{- end}}
    {{- end}}
{{- end}}
{{- range $sd := .ServiceDest}}
    {{- if eq .ReqMode "http" }}
        {{- if ne .Port ""}}
    use_backend {{$.AclName}}-be{{.Port}}_{{.Index}} if url_{{$.AclName}}{{.Port}}_{{.Index}}{{if .ServicePathExclude}} !url_exclude_{{$.AclName}}{{.Port}}_{{.Index}}{{end}}{{if .ServiceDomain}} domain_{{$.AclName}}{{.Port}}_{{.Index}}{{end}}{{if .ServiceHeader}}{{resetIndex}}{{range $key, $value := .ServiceHeader}} hdr_{{$.AclName}}{{$sd.Port}}_{{incIndex}}{{end}}{{end}}{{.SrcPortAclName}}{{ $length := len .UserAgent.Value}}{{if gt $length 0}} user_agent_{{$.AclName}}_{{.UserAgent.AclName}}_{{.Index}}{{end}}
            {{- if $.IsDefaultBackend}}
    default_backend {{$.AclName}}-be{{.Port}}_{{$sd.Index}}
            {{- end}}
        {{- end }}
        {{- if gt $sd.HttpsPort 0 }}
    use_backend https-{{$.AclName}}-be{{.HttpsPort}}_{{.Index}} if url_https_{{$.AclName}}{{.HttpsPort}}_{{.Index}}{{if .ServicePathExclude}} !url_exclude_https_{{$.AclName}}{{.HttpsPort}}_{{.Index}}{{end}}{{if .ServiceDomain}} domain_https_{{$.AclName}}{{.HttpsPort}}_{{.Index}}{{end}}{{.SrcHttpsPortAclName}}{{ $length := len .UserAgent.Value}}{{if gt $length 0}} user_agent_{{$.AclName}}_{{.UserAgent.AclName}}_{{.Index}}{{end}}
        {{- end}}
    {{- end}}
{{- end}}`
	return templateToString(tmplString, s)
}

func getFrontTemplateTcp(servicesByPort map[int]Services) string {
	tmpl := ""
	for _, services := range servicesByPort {
		sort.Sort(services)
		tmplString := `

{{$sd := (index . 0).ServiceDest}}{{$srcPort := (index $sd 0).SrcPort}}frontend tcpFE_{{$srcPort}}
    bind *:{{$srcPort}}
    mode tcp
    {{- if (index . 0).Debug}}{{$debugFormat := (index . 0).DebugFormat}}
    option tcplog
    log global
        {{- if ne $debugFormat ""}}
    log-format {{$debugFormat}}
        {{- end}}
    {{- end}}
    {{- $timeoutClientSd := (index $sd 0).TimeoutClient -}}
    {{- if ne $timeoutClientSd "" }}
    timeout client {{$timeoutClientSd}}s
    {{- end}}
    {{- if (index $sd 0).Clitcpka }}
    option clitcpka
    {{- end}}
    {{- range $s := .}}
        {{- range $sd := .ServiceDest}}
            {{- if $sd.ServiceDomain}}
    acl domain_{{$s.AclName}}{{.Port}}_{{$sd.Index}} {{$s.ServiceDomainAlgo}} -i{{range $sd.ServiceDomain}} {{.}}{{end}}
    use_backend {{$s.AclName}}-be{{$sd.Port}}_{{$sd.Index}} if domain_{{$s.AclName}}{{.Port}}_{{$sd.Index}}
            {{- end}}
            {{- if not $sd.ServiceDomain}}
    default_backend {{$s.AclName}}-be{{$sd.Port}}_{{$sd.Index}}
            {{- end}}
        {{- end}}
    {{- end}}`
		tmpl += templateToString(tmplString, services)
	}
	return tmpl
}

func getFrontTemplateSNI(s Service, si int, genHeader bool) string {
	tmplString := ``
	if genHeader {
		tmplString += fmt.Sprintf(`{{$sd1 := index $.ServiceDest %d}}

frontend service_{{$sd1.SrcPort}}
    bind *:{{$sd1.SrcPort}}
    mode tcp
    {{- if $.Debug}}{{$debugFormat := $.DebugFormat}}
    option tcplog
    log global
        {{- if ne $debugFormat ""}}
    log-format {{$debugFormat}}
        {{- end}}
    {{- end}}
    {{- $timeoutClientSd := $sd1.TimeoutClient -}}
    {{- if ne $timeoutClientSd "" }}
    timeout client {{$timeoutClientSd}}s
    {{- end}}
    {{- if $sd1.Clitcpka }}
    option clitcpka
    {{- end}}
    tcp-request inspect-delay 5s
    tcp-request content accept if { req_ssl_hello_type 1 }`, si)
	}
	tmplString += fmt.Sprintf(`{{$sd := index $.ServiceDest %d}}
    acl sni_{{.AclName}}{{$sd.Port}}-%d{{range $sd.ServicePath}} {{$.PathType}} {{.}}{{end}}
    {{- if ne $sd.SrcPortAcl "" }}
    {{$sd.SrcPortAcl}}
    {{- end }}
    use_backend {{$.ServiceName}}-be{{$sd.Port}}_{{$sd.Index}} if sni_{{$.AclName}}{{$sd.Port}}-%d{{$.AclCondition}}{{$sd.SrcPortAclName}}`, si, si+1, si+1)
	return templateToString(tmplString, s)
}

func getListenTCPGroup(tcpGroups map[string]*tcpGroupInfo) string {
	tmplString := `{{- range $groupName, $info := . }}
{{- $s := $info.TargetService }}
{{- $sd := $info.TargetDest }}

listen tcpListen_{{$groupName}}_{{$sd.SrcPort}}
    bind *:{{$sd.SrcPort}}
    mode tcp
        {{- if $s.Debug}}{{$debugFormat := $s.DebugFormat}}
    option tcplog
    log global
            {{- if ne $debugFormat ""}}
    log-format {{$debugFormat}}
            {{- end}}
        {{- end}}
        {{- if $sd.Clitcpka }}
    option clitcpka
        {{- end}}
        {{- if $sd.CheckTCP}}
    option tcp-check
        {{- end}}
        {{- if ne $sd.TimeoutClient "" }}
    timeout client {{$sd.TimeoutClient}}s
        {{- end}}
        {{- if ne $sd.TimeoutServer ""}}
    timeout server {{ $sd.TimeoutServer }}s
        {{- end}}
        {{- if ne $sd.TimeoutTunnel ""}}
    timeout tunnel {{ $sd.TimeoutTunnel }}s
        {{- end}}
        {{- if ne $sd.BalanceGroup ""}}
    balance {{$sd.BalanceGroup}}
        {{- end}}
        {{- range $tcpIn := .TCPInfo}}
        {{- range $i, $ip := $tcpIn.IPs}}
    server {{$sd.ServiceGroup}}-{{$tcpIn.ServiceName}}{{$tcpIn.Port}}_{{$i}} {{$ip}}:{{$tcpIn.Port}}{{if $sd.CheckTCP}} check{{end}}
        {{- end}}
        {{- end}}
{{- end}}`
	return templateToString(tmplString, tcpGroups)
}

// GetBackTemplate returns template used to create a service backend
// TODO: Change to private function when actions.GetTemplates is moved to the proxy package
// TODO: Create a single string for the template
// TODO: Unify HTTP and HTTPS into a single string
// TODO: Move to a file
func GetBackTemplate(sr *Service) string {
	for i := range sr.ServiceDest {
		if strings.EqualFold(sr.ServiceDest[i].ReqMode, "sni") {
			sr.ServiceDest[i].ReqModeFormatted = "tcp"
		} else {
			sr.ServiceDest[i].ReqModeFormatted = sr.ServiceDest[i].ReqMode
		}
	}
	if len(getSecretOrEnvVar("USERS", "")) > 0 {
		sr.UseGlobalUsers = true
	}
	if strings.EqualFold(os.Getenv("DEBUG"), "true") {
		sr.Debug = true
	}

	// HTTP
	tmpl := `{{- range $sd := .ServiceDest}}
{{- if ne $sd.Port ""}}
{{- if eq $sd.ReqModeFormatted "http" }}
backend {{$.AclName}}-be{{$sd.Port}}_{{.Index}}
    mode {{.ReqModeFormatted}}
    {{- if .HttpsOnly}}
    http-request redirect scheme https{{if .HttpsRedirectCode}} code {{.HttpsRedirectCode}}{{end}} if !{ ssl_fc }
    {{- end}}
    {{- if eq .ReqModeFormatted "http"}}
    http-request add-header X-Forwarded-Proto https if { ssl_fc }
    {{- end}}
    {{- if ne $.ConnectionMode ""}}
    option {{$.ConnectionMode}}
        {{- end}}
        {{- if $.Debug}}
    log global
        {{- end}}`
	tmpl += getHeaders(sr)
	tmpl += `{{- if ne $sd.TimeoutServer ""}}
    timeout server {{ $sd.TimeoutServer }}s
    {{- end}}
    {{- if ne $sd.TimeoutTunnel ""}}
    timeout tunnel {{ $sd.TimeoutTunnel }}s
    {{- end}}
        {{- range $sd.ReqPathSearchReplaceFormatted}}
    http-request set-path %[path,regsub({{.}})]
        {{- end}}
        {{- if eq .VerifyClientSsl true}}
    acl valid_client_cert_{{$.ServiceName}}{{$sd.Port}} ssl_c_used ssl_c_verify 0
    http-request deny unless valid_client_cert_{{$.ServiceName}}{{$sd.Port}}
        {{- end}}
        {{- if .AllowedMethods}}
    acl valid_allowed_method method{{range .AllowedMethods}} {{.}}{{end}}
    http-request deny unless valid_allowed_method
        {{- end}}
        {{- if .DeniedMethods}}
    acl valid_denied_method method{{range .DeniedMethods}} {{.}}{{end}}
    http-request deny if valid_denied_method
        {{- end}}
        {{- if .DenyHttp}}
    http-request deny if !{ ssl_fc }
        {{- end}}
        {{- if eq $.SessionType "sticky-server"}}
    balance roundrobin
    cookie {{$.ServiceName}} insert indirect nocache
        {{- end}}
        {{- range $i, $t := $.Tasks}}
    server {{$.ServiceName}}_{{$i}} {{$t}}:{{$sd.Port}} check cookie {{$.ServiceName}}_{{$i}}{{if eq $sd.SslVerifyNone true}} ssl verify none{{end}}
        {{- end}}
        {{- if not $.Tasks}}
            {{- if eq $.DiscoveryType "DNS"}}
    server-template {{$.ServiceName}} {{$.Replicas}} {{if eq $sd.OutboundHostname ""}}{{$.ServiceName}}{{end}}{{if ne $sd.OutboundHostname ""}}{{$sd.OutboundHostname}}{{end}}:{{$sd.Port}} check{{if eq $.CheckResolvers true}} resolvers docker{{end}}{{if eq $sd.SslVerifyNone true}} ssl verify none{{end}}
            {{- else }}
    server {{$.ServiceName}} {{if eq $sd.OutboundHostname ""}}{{$.ServiceName}}{{end}}{{if ne $sd.OutboundHostname ""}}{{$sd.OutboundHostname}}{{end}}:{{$sd.Port}}{{if eq $.CheckResolvers true}} check resolvers docker{{end}}{{if eq $sd.SslVerifyNone true}} ssl verify none{{end}}
            {{- end}}
        {{- end}}
        {{- if not .IgnoreAuthorization}}
            {{- if and ($.Users) (not .IgnoreAuthorization)}}
    acl {{$.ServiceName}}UsersAcl http_auth({{$.ServiceName}}Users)
    http-request auth realm {{$.ServiceName}}Realm if !{{$.ServiceName}}UsersAcl
            {{- end}}
            {{- if $.UseGlobalUsers}}
    acl defaultUsersAcl http_auth(defaultUsers)
    http-request auth realm defaultRealm if !defaultUsersAcl
            {{- end}}
            {{- if or ($.Users) ($.UseGlobalUsers)}}
    http-request del-header Authorization
            {{- end}}
        {{- end}}
{{- else if eq .ReqModeFormatted "tcp"}}
    {{- if eq $sd.ServiceGroup "" }}
backend {{$.AclName}}-be{{$sd.Port}}_{{.Index}}
    mode tcp
        {{- if .CheckTCP}}
    option tcp-check
        {{- end}}
        {{- if ne $sd.TimeoutServer ""}}
    timeout server {{ $sd.TimeoutServer }}s
        {{- end}}
        {{- if ne $sd.TimeoutTunnel ""}}
    timeout tunnel {{ $sd.TimeoutTunnel }}s
        {{- end}}
    server {{$.ServiceName}} {{if eq $sd.OutboundHostname ""}}{{$.ServiceName}}{{end}}{{if ne $sd.OutboundHostname ""}}{{$sd.OutboundHostname}}{{end}}:{{$sd.Port}}{{if .CheckTCP}} check{{end}}
    {{- end}}
{{- end}}
        {{- if ne $.BackendExtra ""}}
    {{ $.BackendExtra }}
        {{- end}}
{{- end}}
{{- end}}
{{- range $sd := .ServiceDest}}
    {{- if gt $sd.HttpsPort 0}}
backend https-{{$.AclName}}-be{{.HttpsPort}}_{{.Index}}
    mode {{.ReqModeFormatted}}
            {{- if eq .ReqModeFormatted "http"}}
    http-request add-header X-Forwarded-Proto https if { ssl_fc }
            {{- end}}
            {{- if ne $.ConnectionMode ""}}
    option {{$.ConnectionMode}}
            {{- end}}
            {{- if $.Debug}}
    log global
            {{- end}}`
	tmpl += getHeaders(sr)
	tmpl += `
        {{- if ne $sd.TimeoutServer ""}}
    timeout server {{ $sd.TimeoutServer }}s
        {{- end}}
        {{- if ne $sd.TimeoutTunnel ""}}
    timeout tunnel {{ $sd.TimeoutTunnel }}s
        {{- end}}
            {{- range $sd.ReqPathSearchReplaceFormatted}}
    http-request set-path %[path,regsub({{.}})]
            {{- end}}
            {{- if eq .VerifyClientSsl true}}
    acl valid_client_cert_{{$.ServiceName}}{{.HttpsPort}} ssl_c_used ssl_c_verify 0
    http-request deny unless valid_client_cert_{{$.ServiceName}}{{.HttpsPort}}
            {{- end}}
            {{- if .AllowedMethods}}
    acl valid_allowed_method method{{range .AllowedMethods}} {{.}}{{end}}
    http-request deny unless valid_allowed_method
            {{- end}}
            {{- if .DeniedMethods}}
    acl valid_denied_method method{{range .DeniedMethods}} {{.}}{{end}}
    http-request deny if valid_denied_method
            {{- end}}
            {{- if eq $.SessionType "sticky-server"}}
    balance roundrobin
    cookie {{$.ServiceName}} insert indirect nocache
            {{- end}}
            {{- range $i, $t := $.Tasks}}
    server {{$.ServiceName}}_{{$i}} {{$t}}:{{$sd.HttpsPort}} check cookie {{$.ServiceName}}_{{$i}}{{if eq $sd.SslVerifyNone true}} ssl verify none{{end}}
            {{- end}}
            {{- if not $.Tasks}}
                {{- if eq $.DiscoveryType "DNS"}}
    server-template {{$.ServiceName}} {{$.Replicas}} {{if eq $sd.OutboundHostname ""}}{{$.ServiceName}}{{end}}{{if ne $sd.OutboundHostname ""}}{{$sd.OutboundHostname}}{{end}}:{{$sd.HttpsPort}} check{{if eq $.CheckResolvers true}} resolvers docker{{end}}{{if eq $sd.SslVerifyNone true}} ssl verify none{{end}}
                {{- else }}
    server {{$.ServiceName}} {{if eq $sd.OutboundHostname ""}}{{$.ServiceName}}{{end}}{{if ne $sd.OutboundHostname ""}}{{$sd.OutboundHostname}}{{end}}:{{$sd.HttpsPort}}{{if eq $.CheckResolvers true}} check resolvers docker{{end}}{{if eq $sd.SslVerifyNone true}} ssl verify none{{end}}
                {{- end}}
            {{- end}}
            {{- if not .IgnoreAuthorization}}
                {{- if $.Users}}
    acl {{$.ServiceName}}UsersAcl http_auth({{$.ServiceName}}Users)
    http-request auth realm {{$.ServiceName}}Realm if !{{$.ServiceName}}UsersAcl
                {{- end}}
                {{- if $.UseGlobalUsers}}
    acl defaultUsersAcl http_auth(defaultUsers)
    http-request auth realm defaultRealm if !defaultUsersAcl
                {{- end}}
                {{- if or ($.Users) ($.UseGlobalUsers)}}
    http-request del-header Authorization
                {{- end}}
            {{- end}}
            {{- if ne $.BackendExtra ""}}
    {{ $.BackendExtra }}
            {{- end}}
    {{- end}}
{{- end}}`

	return tmpl
}

// FormatServiceForTemplates adds addtional variables to service that is used
// by templates
func FormatServiceForTemplates(sr *Service) {
	if len(sr.AclName) == 0 {
		sr.AclName = sr.ServiceName
	}
	if len(sr.PathType) == 0 {
		sr.PathType = "path_beg"
	}
	if sr.DiscoveryType == "DNS" && sr.Replicas == 0 {
		if ips, err := LookupHost("tasks." + sr.ServiceName); err == nil {
			sr.Replicas = len(ips)
		}
	}
	for i, sd := range sr.ServiceDest {
		if len(sr.ServiceDest[i].ReqMode) == 0 {
			sr.ServiceDest[i].ReqMode = "http"
		}

		srcPort := sd.SrcPort
		if sd.HttpsPort > 0 && srcPort == 0 {
			srcPort = 80
		}
		if srcPort > 0 {
			sr.ServiceDest[i].SrcPortAclName = fmt.Sprintf(" srcPort_%s%d_%d", sr.AclName, srcPort, sd.Index)
			sr.ServiceDest[i].SrcPortAcl = fmt.Sprintf("acl srcPort_%s%d_%d dst_port %d",
				sr.AclName, srcPort, sd.Index, srcPort)
		}
		if srcPort > 0 && (len(sd.Port) > 0 || len(sd.ServicePathExclude) > 0) {
			sr.ServiceDest[i].IncludeSrcPortACL = true
		}
		sr.ServiceDest[i].SrcPort = srcPort

		// Handle https port
		srcHttpsPort := sd.SrcHttpsPort
		if sd.HttpsPort > 0 && srcHttpsPort == 0 {
			srcHttpsPort = 443
		}
		if srcHttpsPort > 0 {
			sr.ServiceDest[i].SrcHttpsPortAclName = fmt.Sprintf(" srcHttpsPort_%s%d_%d", sr.AclName, srcHttpsPort, sd.Index)
			sr.ServiceDest[i].SrcHttpsPortAcl = fmt.Sprintf("acl srcHttpsPort_%s%d_%d dst_port %d", sr.AclName, srcHttpsPort, sd.Index, srcHttpsPort)
		}

		if srcHttpsPort > 0 && sd.HttpsPort > 0 {
			sr.ServiceDest[i].IncludeSrcHttpsPortACL = true
		}
		sr.ServiceDest[i].SrcHttpsPort = srcHttpsPort

	}
}

func getHeaders(sr *Service) string {
	tmpl := ""
	for _, header := range sr.AddReqHeader {
		tmpl += fmt.Sprintf(`
    http-request add-header %s`,
			header,
		)
	}
	for _, header := range sr.SetReqHeader {
		tmpl += fmt.Sprintf(`
    http-request set-header %s`,
			header,
		)
	}
	for _, header := range sr.AddResHeader {
		tmpl += fmt.Sprintf(`
    http-response add-header %s`,
			header,
		)
	}
	for _, header := range sr.SetResHeader {
		tmpl += fmt.Sprintf(`
    http-response set-header %s`,
			header,
		)
	}
	for _, header := range sr.DelReqHeader {
		tmpl += fmt.Sprintf(`
    http-request del-header %s`,
			header,
		)
	}
	for _, header := range sr.DelResHeader {
		tmpl += fmt.Sprintf(`
    http-response del-header %s`,
			header,
		)
	}
	return tmpl
}

func templateToString(templateString string, data interface{}) string {
	i := -1
	funcMap := template.FuncMap{
		"resetIndex": func() string {
			i = -1
			return ""
		},
		"incIndex": func() int {
			i += 1
			return i
		},
	}
	tmpl, _ := template.New("template").Funcs(funcMap).Parse(templateString)
	var b bytes.Buffer
	tmpl.Execute(&b, data)
	return b.String()
}

func putDomainAlgo(s *Service) {
	if len(s.ServiceDomainAlgo) == 0 {
		s.ServiceDomainAlgo = os.Getenv("SERVICE_DOMAIN_ALGO")
	}
	for _, sd := range s.ServiceDest {
		for i, domain := range sd.ServiceDomain {
			if strings.HasPrefix(domain, "*") {
				sd.ServiceDomain[i] = strings.Trim(domain, "*")
				s.ServiceDomainAlgo = "hdr_end(host)"
				return
			}
		}
	}
}
