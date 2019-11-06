package docker

import (
	"context"
	"net/http"

	"github.com/docker/docker/client"

	"github.com/portainer/portainer/api/http/proxy/factory/responseutils"

	"github.com/portainer/portainer/api"
)

const (
	secretObjectIdentifier = "ID"
)

func getInheritedResourceControlFromSecretLabels(dockerClient *client.Client, secretID string, resourceControls []portainer.ResourceControl) (*portainer.ResourceControl, error) {
	secret, _, err := dockerClient.SecretInspectWithRaw(context.Background(), secretID)
	if err != nil {
		return nil, err
	}

	swarmStackName := secret.Spec.Labels[resourceLabelForDockerSwarmStackName]
	if swarmStackName != "" {
		return portainer.GetResourceControlByResourceIDAndType(swarmStackName, portainer.StackResourceControl, resourceControls), nil
	}

	return nil, nil
}

// secretListOperation extracts the response as a JSON object, loop through the secrets array
// decorate and/or filter the secrets based on resource controls before rewriting the response
func (transport *Transport) secretListOperation(response *http.Response, executor *operationExecutor) error {
	// SecretList response is a JSON array
	// https://docs.docker.com/engine/api/v1.28/#operation/SecretList
	responseArray, err := responseutils.GetResponseAsJSONArray(response)
	if err != nil {
		return err
	}

	resourceOperationParameters := &resourceOperationParameters{
		secretObjectIdentifier,
		portainer.SecretResourceControl,
		selectorSecretLabels,
	}

	responseArray, err = transport.applyAccessControlOnResourceList(resourceOperationParameters, responseArray, executor)
	if err != nil {
		return err
	}

	return responseutils.RewriteResponse(response, responseArray, http.StatusOK)
}

// secretInspectOperation extracts the response as a JSON object, verify that the user
// has access to the secret based on resource control (check are done based on the secretID and optional Swarm service ID)
// and either rewrite an access denied response or a decorated secret.
func (transport *Transport) secretInspectOperation(response *http.Response, executor *operationExecutor) error {
	// SecretInspect response is a JSON object
	// https://docs.docker.com/engine/api/v1.28/#operation/SecretInspect
	responseObject, err := responseutils.GetResponseAsJSONOBject(response)
	if err != nil {
		return err
	}

	resourceOperationParameters := &resourceOperationParameters{
		secretObjectIdentifier,
		portainer.SecretResourceControl,
		selectorSecretLabels,
	}

	return transport.applyAccessControlOnResource(resourceOperationParameters, responseObject, response, executor)
}

// selectorSecretLabels retrieve the Labels of the secret if present.
// Secret schema references:
// https://docs.docker.com/engine/api/v1.40/#operation/SecretList
// https://docs.docker.com/engine/api/v1.40/#operation/SecretInspect
func selectorSecretLabels(responseObject map[string]interface{}) map[string]interface{} {
	// Labels are stored under Spec.Labels
	secretSpec := responseutils.GetJSONObject(responseObject, "Spec")
	if secretSpec != nil {
		secretLabelsObject := responseutils.GetJSONObject(secretSpec, "Labels")
		return secretLabelsObject
	}
	return nil
}
