package cspbuilder

import (
	"sort"
	"strings"
)

const (
	// Fetch Directives.
	ChildSrc      = "child-src"
	ConnectSrc    = "connect-src"
	DefaultSrc    = "default-src"
	FontSrc       = "font-src"
	FrameSrc      = "frame-src"
	ImgSrc        = "img-src"
	ManifestSrc   = "manifest-src"
	MediaSrc      = "media-src"
	ObjectSrc     = "object-src"
	PrefetchSrc   = "prefetch-src"
	ScriptSrc     = "script-src"
	ScriptSrcAttr = "script-src-attr"
	ScriptSrcElem = "script-src-elem"
	StyleSrc      = "style-src"
	StyleSrcAttr  = "style-src-attr"
	StyleSrcElem  = "style-src-elem"
	WorkerSrc     = "worker-src"

	// Document Directives.
	BaseURI = "base-uri"
	Sandbox = "sandbox"

	// Navigation directives.
	FormAction     = "form-action"
	FrameAncestors = "frame-ancestors"
	NavigateTo     = "navigate-to"

	// Reporting directives.
	ReportURI = "report-uri"
	ReportTo  = "report-to"

	// Other directives.
	RequireTrustedTypesFor  = "require-trusted-types-for"
	TrustedTypes            = "trusted-types"
	UpgradeInsecureRequests = "upgrade-insecure-requests"
)

type Builder struct {
	Directives map[string]([]string)
}

// MustBuild is like Build but panics if an error occurs.
func (builder *Builder) MustBuild() string {
	policy, err := builder.Build()
	if err != nil {
		panic(err)
	}

	return policy
}

// Build creates a content security policy string from the specified directives.
// If any directive contains invalid values, an error is returned instead.
func (builder *Builder) Build() (string, error) {
	var sb strings.Builder

	// Pull the directive keys out.
	directiveKeys := []string{}
	for key := range builder.Directives {
		directiveKeys = append(directiveKeys, key)
	}

	// Sort the policies: https://www.w3.org/TR/CSP3/#framework-policy
	sort.Strings(directiveKeys)

	for _, directive := range directiveKeys {
		if sb.Len() > 0 {
			sb.WriteString("; ")
		}

		switch directive {
		case Sandbox:
			err := buildDirectiveSandbox(&sb, builder.Directives[directive])
			if err != nil {
				return "", err
			}
		case FrameAncestors:
			err := buildDirectiveFrameAncestors(&sb, builder.Directives[directive])
			if err != nil {
				return "", err
			}
		case ReportTo:
			err := buildDirectiveReportTo(&sb, builder.Directives[directive])
			if err != nil {
				return "", err
			}
		case RequireTrustedTypesFor:
			err := buildDirectiveRequireTrustedTypesFor(&sb, builder.Directives[directive])
			if err != nil {
				return "", err
			}
		case TrustedTypes:
			err := buildDirectiveTrustedTypes(&sb, builder.Directives[directive])
			if err != nil {
				return "", err
			}
		case UpgradeInsecureRequests:
			err := buildDirectiveUpgradeInsecureRequests(&sb, builder.Directives[directive])
			if err != nil {
				return "", err
			}
		default:
			// no special handling of directive values needed
			err := buildDirectiveDefault(&sb, directive, builder.Directives[directive])
			if err != nil {
				return "", err
			}
		}
	}

	return sb.String(), nil
}
