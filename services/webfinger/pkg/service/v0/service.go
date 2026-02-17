package service

import (
	"context"
	"net/url"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/config"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/webfinger"
)

// Service defines the extension handlers.
type Service interface {
	// Webfinger is the endpoint for retrieving various href relations.
	//
	//	GET /.well-known/webfinger?
	//	     resource=acct%3Acarol%40example.com&
	//	     rel=http%3A%2F%2Fwebfinger.opencloud%rel%2Fserver-instance
	//	     HTTP/1.1
	//	Host: example.com
	//
	// The server might respond like this:
	//
	//	HTTP/1.1 200 OK
	//	Access-Control-Allow-Origin: *
	//	Content-Type: application/jrd+json
	//
	//	{
	//	  "subject" : "acct:carol@example.com",
	//	  "links" :
	//	  [
	//	    {
	//	      "rel" : "http://webfinger.opencloud/rel/server-instance",
	//	      "href" : "https://instance.example.com",
	//	      "titles": {
	//	        "en": "Readable Instance Name"
	//	      }
	//	    },
	//	    {
	//	      "rel" : "http://webfinger.opencloud/rel/server-instance",
	//	      "href" : "https://otherinstance.example.com",
	//	      "titles": {
	//	        "en": "Other Readable Instance Name"
	//	      }
	//	    }
	//	  ]
	//	}
	Webfinger(ctx context.Context, queryTarget *url.URL, rels []string, platform string) (webfinger.JSONResourceDescriptor, error)
}

type RelationProvider interface {
	Add(ctx context.Context, platform string, jrd *webfinger.JSONResourceDescriptor)
}

// New returns a new instance of Service
func New(opts ...Option) (Service, error) {
	options := newOptions(opts...)

	// TODO use fallback implementations of InstanceIdLookup and InstanceLookup?
	// The InstanceIdLookup may have to happen earlier?

	return svc{
		log:               options.Logger,
		config:            options.Config,
		relationProviders: options.RelationProviders,
	}, nil
}

type svc struct {
	config            *config.Config
	log               log.Logger
	relationProviders map[string]RelationProvider
}

// TODO implement different implementations:
// static one returning the href or a configurable domain
// regex one returning different instances based on the regex that matches
// claim one that reads a claim and then fetches the instance?
// that is actually two interfaces / steps:
// - one that determines the instances/schools id (read from claim, regex match)
// - one that looks up in instance by id (use template, read from json, read from ldap, read from graph)

// Webfinger implements the service interface
func (s svc) Webfinger(ctx context.Context, queryTarget *url.URL, rel []string, platform string) (webfinger.JSONResourceDescriptor, error) {

	jrd := webfinger.JSONResourceDescriptor{
		Subject: queryTarget.String(),
	}

	if len(rel) == 0 {
		// add all configured relation providers
		for _, relation := range s.relationProviders {
			relation.Add(ctx, platform, &jrd)
		}
	} else {
		// only add requested relations
		for _, r := range rel {
			if relation, ok := s.relationProviders[r]; ok {
				relation.Add(ctx, platform, &jrd)
			}
		}
	}

	return jrd, nil
}
