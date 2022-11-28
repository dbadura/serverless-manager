package v1alpha1

import "k8s.io/utils/pointer"

const (
	defaultInternalServerAddress = "k3d-kyma-registry:5000"
	defaultRegistryAddress       = "k3d-kyma-registry:5000"
	defaultServerAddress         = "k3d-kyma-registry:5000"
	defaultGateway               = "kyma-system/kyma-gateway"
	defaultGatewayCert           = "kyma-gateway-certs"
)

// TODO: refactor - we don't want to have method full of ifs
func (s *ServerlessSpec) Default() {

	// if DockerRegistry struct is nil configure use of k3d registry
	if s.DockerRegistry == nil {
		s.DockerRegistry = newK3DDockerRegistry()
	}
}

func newK3DDockerRegistry() *DockerRegistry {
	return &DockerRegistry{
		EnableInternal:        pointer.Bool(false),
		InternalServerAddress: pointer.String(defaultInternalServerAddress),
		RegistryAddress:       pointer.String(defaultRegistryAddress),
		ServerAddress:         pointer.String(defaultServerAddress),
		Gateway:               pointer.String(defaultGateway),
		GatewayCert:           pointer.String(defaultGatewayCert),
	}
}
