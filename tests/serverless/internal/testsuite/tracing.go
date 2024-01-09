package testsuite

import (
	"fmt"
	"github.com/kyma-project/serverless/tests/serverless/internal"
	"github.com/kyma-project/serverless/tests/serverless/internal/assertion"
	"github.com/kyma-project/serverless/tests/serverless/internal/executor"
	"github.com/kyma-project/serverless/tests/serverless/internal/resources/app"
	"github.com/kyma-project/serverless/tests/serverless/internal/resources/function"
	"github.com/kyma-project/serverless/tests/serverless/internal/resources/namespace"
	"github.com/kyma-project/serverless/tests/serverless/internal/resources/runtimes"
	"github.com/kyma-project/serverless/tests/serverless/internal/utils"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"time"

	serverlessv1alpha2 "github.com/kyma-project/serverless/components/serverless/pkg/apis/serverless/v1alpha2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

const (
	HTTPAppName  = "http-server"
	HTTPAppImage = "nginx"
)

func FunctionTracingTest(restConfig *rest.Config, cfg internal.Config, logf *logrus.Entry) (executor.Step, error) {
	now := time.Now()
	cfg.Namespace = fmt.Sprintf("%s-%02dh%02dm%02ds", "test-tracing", now.Hour(), now.Minute(), now.Second())

	dynamicCli, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "while creating dynamic client")
	}

	coreCli, err := typedcorev1.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "while creating k8s CoreV1Client")
	}

	appsCli, err := typedappsv1.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "while creating k8s apps client")
	}

	python39Logger := logf.WithField(runtimeKey, "python39")
	nodejs16Logger := logf.WithField(runtimeKey, "nodejs16")
	nodejs18Logger := logf.WithField(runtimeKey, "nodejs18")

	genericContainer := utils.Container{
		DynamicCli:  dynamicCli,
		Namespace:   cfg.Namespace,
		WaitTimeout: cfg.WaitTimeout,
		Verbose:     cfg.Verbose,
		Log:         logf,
	}

	python39Fn := function.NewFunction("python39", genericContainer.Namespace, cfg.KubectlProxyEnabled, genericContainer.WithLogger(python39Logger))

	nodejs16Fn := function.NewFunction("nodejs16", genericContainer.Namespace, cfg.KubectlProxyEnabled, genericContainer.WithLogger(nodejs16Logger))

	nodejs18Fn := function.NewFunction("nodejs18", genericContainer.Namespace, cfg.KubectlProxyEnabled, genericContainer.WithLogger(nodejs18Logger))

	logf.Infof("Testing function in namespace: %s", cfg.Namespace)

	httpAppURL, err := utils.GetSvcURL(HTTPAppName, genericContainer.Namespace, false)
	if err != nil {
		return nil, errors.Wrap(err, "while creating http application URL")
	}

	poll := utils.Poller{
		MaxPollingTime:     cfg.MaxPollingTime,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		DataKey:            internal.TestDataKey,
	}
	return executor.NewSerialTestRunner(logf, "Runtime test",
		namespace.NewNamespaceStep(logf, fmt.Sprintf("Create %s namespace", genericContainer.Namespace), genericContainer.Namespace, coreCli),
		app.NewApplication("Create HTTP basic application", HTTPAppName, HTTPAppImage, int32(80), appsCli.Deployments(genericContainer.Namespace), coreCli.Services(genericContainer.Namespace), genericContainer),
		executor.NewParallelRunner(logf, "Fn tests",
			executor.NewSerialTestRunner(python39Logger, "Python39 test",
				function.CreateFunction(python39Logger, python39Fn, "Create Python39 Function", runtimes.BasicTracingPythonFunction(serverlessv1alpha2.Python39, httpAppURL.String())),
				assertion.TracingHTTPCheck(python39Logger, "Python39 tracing headers check", python39Fn.FunctionURL, poll),
			),
			executor.NewSerialTestRunner(nodejs16Logger, "NodeJS16 test",
				function.CreateFunction(nodejs16Logger, nodejs16Fn, "Create NodeJS16 Function", runtimes.BasicTracingNodeFunction(serverlessv1alpha2.NodeJs16, httpAppURL.String())),
				assertion.TracingHTTPCheck(nodejs16Logger, "NodeJS16 tracing headers check", nodejs16Fn.FunctionURL, poll),
			),
			executor.NewSerialTestRunner(nodejs18Logger, "NodeJS18 test",
				function.CreateFunction(nodejs18Logger, nodejs18Fn, "Create NodeJS18 Function", runtimes.BasicTracingNodeFunction(serverlessv1alpha2.NodeJs18, httpAppURL.String())),
				assertion.TracingHTTPCheck(nodejs18Logger, "NodeJS18 tracing headers check", nodejs18Fn.FunctionURL, poll),
			),
		),
	), nil
}
