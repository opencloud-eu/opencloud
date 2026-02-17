package relations

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"text/template"

	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/config"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/service/v0"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/webfinger"
)

const (
	OpenCloudInstanceRel = "http://webfinger.opencloud/rel/server-instance"
)

type compiledInstance struct {
	config.Instance
	compiledRegex *regexp.Regexp
	hrefTemplate  *template.Template
}

type openCloudInstance struct {
	instances    []compiledInstance
	openCloudURL string
	instanceHost string
}

// OpenCloudInstance adds one or more OpenCloud instance relations
func OpenCloudInstance(instances []config.Instance, openCloudURL string) (service.RelationProvider, error) {
	compiledInstances := make([]compiledInstance, 0, len(instances))
	var err error
	for _, instance := range instances {
		compiled := compiledInstance{Instance: instance}
		compiled.compiledRegex, err = regexp.Compile(instance.Regex)
		if err != nil {
			return nil, err
		}
		compiled.hrefTemplate, err = template.New(instance.Claim + ":" + instance.Regex + ":" + instance.Href).Parse(instance.Href)
		if err != nil {
			return nil, err
		}
		compiledInstances = append(compiledInstances, compiled)
	}

	u, err := url.Parse(openCloudURL)
	if err != nil {
		return nil, err
	}
	return &openCloudInstance{
		instances:    compiledInstances,
		openCloudURL: openCloudURL,
		instanceHost: u.Host + u.Path,
	}, nil
}

func (l *openCloudInstance) Add(ctx context.Context, _ string, jrd *webfinger.JSONResourceDescriptor) {
	if jrd == nil {
		jrd = &webfinger.JSONResourceDescriptor{}
	}
	if claims := oidc.FromContext(ctx); claims != nil {
		if value, ok := claims[oidc.PreferredUsername].(string); ok {
			jrd.Subject = "acct:" + value + "@" + l.instanceHost
		} else if value, ok := claims[oidc.Email].(string); ok {
			jrd.Subject = "mailto:" + value
		}
		// allow referencing OC_URL in the template
		claims["OC_URL"] = l.openCloudURL
		for _, instance := range l.instances {
			if value, ok := claims[instance.Claim].(string); ok && instance.compiledRegex.MatchString(value) {
				var tmplWriter strings.Builder
				instance.hrefTemplate.Execute(&tmplWriter, claims)
				jrd.Links = append(jrd.Links, webfinger.Link{
					Rel:    OpenCloudInstanceRel,
					Href:   tmplWriter.String(),
					Titles: instance.Titles,
				})
				if instance.Break {
					break
				}
			}
		}
	}
}
