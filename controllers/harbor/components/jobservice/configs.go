package jobservice

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	goharborv1alpha1 "github.com/goharbor/harbor-operator/api/v1alpha1"
	"github.com/goharbor/harbor-operator/pkg/factories/application"
	"github.com/markbates/pkger"
	"github.com/pkg/errors"
)

const (
	configName = "config.yaml"
)

const (
	logsDirectory = "/var/log/jobs"
)

var (
	once         sync.Once
	config       []byte
	hookMaxRetry = 5
)

func InitConfigMaps() {
	file, err := pkger.Open("/assets/templates/jobservice/config.yaml")
	if err != nil {
		panic(errors.Wrapf(err, "cannot open JobService configuration template %s", "/assets/templates/jobservice/config.yaml"))
	}
	defer file.Close()

	config, err = ioutil.ReadAll(file)
	if err != nil {
		panic(errors.Wrapf(err, "cannot read JobService configuration template %s", "/assets/templates/jobservice/config.yaml"))
	}
}

func (j *JobService) GetConfigMaps(ctx context.Context) []*corev1.ConfigMap {
	once.Do(InitConfigMaps)

	operatorName := application.GetName(ctx)
	harborName := j.harbor.Name

	return []*corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      j.harbor.NormalizeComponentName(goharborv1alpha1.JobServiceName),
				Namespace: j.harbor.Namespace,
				Labels: map[string]string{
					"app":      goharborv1alpha1.JobServiceName,
					"harbor":   harborName,
					"operator": operatorName,
				},
			},
			BinaryData: map[string][]byte{
				configName: config,
			},
			Data: map[string]string{
				"REGISTRY_CONTROLLER_URL":          fmt.Sprintf("http://%s:8080", j.harbor.NormalizeComponentName(goharborv1alpha1.RegistryName)),
				"JOBSERVICE_WEBHOOK_JOB_MAX_RETRY": fmt.Sprintf("%d", hookMaxRetry),
				"JOB_SERVICE_POOL_WORKERS":         fmt.Sprintf("%d", j.harbor.Spec.Components.JobService.WorkerCount),
			},
		},
	}
}

func (j *JobService) GetConfigMapsCheckSum() string {
	value := fmt.Sprintf("%d\n%d\n%x", hookMaxRetry, j.harbor.Spec.Components.JobService.WorkerCount, config)
	sum := sha256.New().Sum([]byte(value))

	return fmt.Sprintf("%x", sum)
}
