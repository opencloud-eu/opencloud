//go:build enable_vips

package parity

// builtinBuild is the resize implementation compiled into this run, and the
// override key for a divergence only it has. The container images ship vips.
const builtinBuild = "vips"
