package kubernetes

import (
	"fmt"
	"github.com/kubernetes/kompose/pkg/kobject"
	"k8s.io/kubernetes/pkg/api"
)

// ConfigLogVolumes configure the volumes for log.
func (k *Kubernetes) ConfigLogVolumes(name string, service kobject.ServiceConfig) ([]api.VolumeMount, []api.Volume) {
	//initializing volumemounts and volumes
	volumeMounts := []api.VolumeMount{}
	volumes := []api.Volume{}

	for logStore, volume := range service.LogVolumes {
		//naming volumes if multiple tmpfs are provided
		volumeName := fmt.Sprintf("volumn-sls-%s", logStore)
		// create a new volume mount object and append to list
		volMount := api.VolumeMount{
			Name:      volumeName,
			MountPath: volume,
		}
		volumeMounts = append(volumeMounts, volMount)

		//create specific empty volumes
		volSource := k.ConfigEmptyVolumeSource(volumeName)

		// create a new volume object using the volsource and add to list
		vol := api.Volume{
			Name:         volumeName,
			VolumeSource: *volSource,
		}
		volumes = append(volumes, vol)
	}
	return volumeMounts, volumes
}
