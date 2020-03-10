package e2e

import (
	"context"
	"github.com/Juniper/contrail-operator/pkg/client/swift"
	"net/http"
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	contrail "github.com/Juniper/contrail-operator/pkg/apis/contrail/v1alpha1"
	"github.com/Juniper/contrail-operator/pkg/client/keystone"
	"github.com/Juniper/contrail-operator/pkg/client/kubeproxy"
	"github.com/Juniper/contrail-operator/test/logger"
	wait "github.com/Juniper/contrail-operator/test/wait"
)

func TestCommandServices(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()
	log := logger.New(t, "contrail", test.Global.Client)

	if err := test.AddToFrameworkScheme(contrail.SchemeBuilder.AddToScheme, &contrail.ManagerList{}); err != nil {
		t.Fatalf("Failed to add framework scheme: %v", err)
	}

	if err := ctx.InitializeClusterResources(&test.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval}); err != nil {
		t.Fatalf("Failed to initialize cluster resources: %v", err)
	}
	namespace, err := ctx.GetNamespace()
	assert.NoError(t, err)
	f := test.Global
	proxy, err := kubeproxy.New(f.KubeConfig)
	require.NoError(t, err)

	t.Run("given contrail-operator is running", func(t *testing.T) {
		err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "contrail-operator", 1, retryInterval, waitTimeout)
		if err != nil {
			log.DumpPods()
		}
		assert.NoError(t, err)

		trueVal := true
		oneVal := int32(1)

		psql := &contrail.Postgres{
			ObjectMeta: meta.ObjectMeta{Namespace: namespace, Name: "commandtest-psql"},
			Spec: contrail.PostgresSpec{
				Containers: map[string]*contrail.Container{
					"postgres": {Image: "registry:5000/postgres"},
				},
			},
		}

		memcached := &contrail.Memcached{
			ObjectMeta: meta.ObjectMeta{
				Namespace: namespace,
				Name:      "commandtest-memcached",
			},
			Spec: contrail.MemcachedSpec{
				ServiceConfiguration: contrail.MemcachedConfiguration{
					Container: contrail.Container{Image: "registry:5000/centos-binary-memcached:master"},
				},
			},
		}

		keystoneResource := &contrail.Keystone{
			ObjectMeta: meta.ObjectMeta{Namespace: namespace, Name: "commandtest-keystone"},
			Spec: contrail.KeystoneSpec{
				CommonConfiguration: contrail.CommonConfiguration{HostNetwork: &trueVal},
				ServiceConfiguration: contrail.KeystoneConfiguration{
					MemcachedInstance:  "commandtest-memcached",
					PostgresInstance:   "commandtest-psql",
					ListenPort:         5555,
					KeystoneSecretName: "commandtest-keystone-adminpass-secret",
					Containers: map[string]*contrail.Container{
						"keystoneDbInit": {Image: "registry:5000/postgresql-client"},
						"keystoneInit":   {Image: "registry:5000/centos-binary-keystone:master"},
						"keystone":       {Image: "registry:5000/centos-binary-keystone:master"},
						"keystoneSsh":    {Image: "registry:5000/centos-binary-keystone-ssh:master"},
						"keystoneFernet": {Image: "registry:5000/centos-binary-keystone-fernet:master"},
					},
				},
			},
		}

		swiftInstance := &contrail.Swift{
			ObjectMeta: meta.ObjectMeta{
				Namespace: namespace,
				Name:      "commandtest-swift",
			},
			Spec: contrail.SwiftSpec{
				ServiceConfiguration: contrail.SwiftConfiguration{
					Containers: map[string]*contrail.Container{
						"ring-reconciler": {Image: "registry:5000/centos-source-swift-base:master"},
					},
					SwiftStorageConfiguration: contrail.SwiftStorageConfiguration{
						AccountBindPort:   6001,
						ContainerBindPort: 6002,
						ObjectBindPort:    6000,
						Device:            "d1",
						Containers: map[string]*contrail.Container{
							"swiftObjectExpirer":       {Image: "registry:5000/centos-binary-swift-object-expirer:master"},
							"swiftObjectUpdater":       {Image: "registry:5000/centos-binary-swift-object:master"},
							"swiftObjectReplicator":    {Image: "registry:5000/centos-binary-swift-object:master"},
							"swiftObjectAuditor":       {Image: "registry:5000/centos-binary-swift-object:master"},
							"swiftObjectServer":        {Image: "registry:5000/centos-binary-swift-object:master"},
							"swiftContainerUpdater":    {Image: "registry:5000/centos-binary-swift-container:master"},
							"swiftContainerReplicator": {Image: "registry:5000/centos-binary-swift-container:master"},
							"swiftContainerAuditor":    {Image: "registry:5000/centos-binary-swift-container:master"},
							"swiftContainerServer":     {Image: "registry:5000/centos-binary-swift-container:master"},
							"swiftAccountReaper":       {Image: "registry:5000/centos-binary-swift-account:master"},
							"swiftAccountReplicator":   {Image: "registry:5000/centos-binary-swift-account:master"},
							"swiftAccountAuditor":      {Image: "registry:5000/centos-binary-swift-account:master"},
							"swiftAccountServer":       {Image: "registry:5000/centos-binary-swift-account:master"},
						},
					},
					SwiftProxyConfiguration: contrail.SwiftProxyConfiguration{
						MemcachedInstance:  "commandtest-memcached",
						ListenPort:         5080,
						KeystoneInstance:   "commandtest-keystone",
						SwiftPassword:      "swiftpass",
						KeystoneSecretName: "commandtest-keystone-adminpass-secret",
						Containers: map[string]*contrail.Container{
							"init": {Image: "registry:5000/centos-binary-kolla-toolbox:master"},
							"api":  {Image: "registry:5000/centos-binary-swift-proxy-server:master"},
						},
					},
				},
			},
		}

		command := &contrail.Command{
			ObjectMeta: meta.ObjectMeta{
				Name: "commandtest",
			},
			Spec: contrail.CommandSpec{
				CommonConfiguration: contrail.CommonConfiguration{
					Activate:    &trueVal,
					Create:      &trueVal,
					HostNetwork: &trueVal,
				},
				ServiceConfiguration: contrail.CommandConfiguration{
					PostgresInstance:   "commandtest-psql",
					KeystoneSecretName: "commandtest-keystone-adminpass-secret",
					ConfigAPIURL:       "https://kind-control-plane:8082",
					TelemetryURL:       "https://kind-control-plane:8081",
					KeystoneInstance:   "commandtest-keystone",
					SwiftInstance:      "commandtest-swift",
					Containers: map[string]*contrail.Container{
						"init": {Image: "registry:5000/contrail-command:1912-latest"},
						"api":  {Image: "registry:5000/contrail-command:1912-latest"},
					},
				},
			},
		}

		cluster := &contrail.Manager{
			ObjectMeta: meta.ObjectMeta{
				Name:      "cluster1",
				Namespace: namespace,
			},
			Spec: contrail.ManagerSpec{
				CommonConfiguration: contrail.CommonConfiguration{
					Replicas:    &oneVal,
					HostNetwork: &trueVal,
				},
				Services: contrail.Services{
					Postgres:  psql,
					Keystone:  keystoneResource,
					Memcached: memcached,
					Command:   command,
					Swift:     swiftInstance,
				},
				KeystoneSecretName: "commandtest-keystone-adminpass-secret",
			},
		}

		adminPassWordSecret := &core.Secret{
			ObjectMeta: meta.ObjectMeta{
				Name:      "commandtest-keystone-adminpass-secret",
				Namespace: namespace,
			},
			StringData: map[string]string{
				"password": "test123",
			},
		}

		t.Run("when manager resource with command and dependencies is created", func(t *testing.T) {
			err = f.Client.Create(context.TODO(), adminPassWordSecret, &test.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
			assert.NoError(t, err)

			err = f.Client.Create(context.TODO(), cluster, &test.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
			assert.NoError(t, err)

			w := wait.Wait{
				Namespace:     namespace,
				Timeout:       waitTimeout,
				RetryInterval: retryInterval,
				KubeClient:    f.KubeClient,
				Logger:        log,
			}

			t.Run("then a ready Command Deployment should be created", func(t *testing.T) {
				assert.NoError(t, w.ForDeployment("commandtest-command-deployment"))
			})

			t.Run("then a ready Keystone StatefulSet should be created", func(t *testing.T) {
				assert.NoError(t, w.ForReadyStatefulSet("commandtest-keystone-keystone-statefulset"))
			})

			t.Run("then Swift should become active", func(t *testing.T) {
				err := wait.Contrail{
					Namespace:     namespace,
					Timeout:       5 * time.Minute,
					RetryInterval: retryInterval,
					Client:        f.Client,
				}.ForSwiftActive(command.Spec.ServiceConfiguration.SwiftInstance)
				require.NoError(t, err)
			})

			var commandPods *core.PodList
			var err error
			t.Run("then a ready Command deployment pod should be created", func(t *testing.T) {
				commandPods, err = f.KubeClient.CoreV1().Pods("contrail").List(meta.ListOptions{
					LabelSelector: "command=commandtest",
				})
				assert.NoError(t, err)
				assert.NotEmpty(t, commandPods.Items)
			})

			commandProxy := proxy.NewClientWithPath("contrail", commandPods.Items[0].Name, 9091, "/keystone")
			keystoneClient := keystone.NewClient(commandProxy)

			t.Run("then the local keystone service should handle request for a token", func(t *testing.T) {
				_, err := keystoneClient.PostAuthTokens("admin", "test123", "admin")
				assert.NoError(t, err)
			})

			t.Run("then the proxied keystone service should handle request for a token", func(t *testing.T) {
				headers := http.Header{}
				headers.Set("X-Cluster-ID", "53494ca8-f40c-11e9-83ae-38c986460fd4")
				_, err = keystoneClient.PostAuthTokensWithHeaders("admin", "test123", "admin", headers)
				assert.NoError(t, err)
			})

			var swiftProxyPods *core.PodList
			swiftProxyLabel := "SwiftProxy=" + command.Spec.ServiceConfiguration.SwiftInstance + "-proxy"
			swiftProxyPods, err = f.KubeClient.CoreV1().Pods("contrail").List(meta.ListOptions{
				LabelSelector: swiftProxyLabel,
			})
			assert.NoError(t, err)
			require.NotEmpty(t, swiftProxyPods.Items)
			keystoneProxy := proxy.NewClient("contrail", command.Spec.ServiceConfiguration.KeystoneInstance+"-keystone-statefulset-0", 5555)
			keystoneClient = keystone.NewClient(keystoneProxy)
			tokens, _ := keystoneClient.PostAuthTokens("admin", string(adminPassWordSecret.Data["password"]), "admin")
			swiftProxyPod := swiftProxyPods.Items[0].Name
			swiftProxy := proxy.NewClient("contrail", swiftProxyPod, 5080)
			swiftURL := tokens.EndpointURL("swift", "public")
			swiftClient, err := swift.NewClient(swiftProxy, tokens.XAuthTokenHeader, swiftURL)
			require.NoError(t, err)

			t.Run("then swift container should be created", func(t *testing.T) {
				err = swiftClient.GetContainer("contrail_container")
				assert.NoError(t, err)
			})

			t.Run("and when a file is put to the created container", func(t *testing.T) {
				err = swiftClient.PutFile("contrail_container", "test-file", []byte("payload"))
				require.NoError(t, err)

				t.Run("then the file can be downloaded without authentication and has proper payload", func(t *testing.T) {
					swiftNoAuthClient, err := swift.NewClient(swiftProxy, "", swiftURL)
					require.NoError(t, err)
					contents, err := swiftNoAuthClient.GetFile("contrail_container", "test-file")
					require.NoError(t, err)
					assert.Equal(t, "payload", string(contents))
				})
			})
		})

		t.Run("when reference cluster is deleted", func(t *testing.T) {
			pp := meta.DeletePropagationForeground
			err = f.Client.Delete(context.TODO(), cluster, &client.DeleteOptions{
				PropagationPolicy: &pp,
			})
			assert.NoError(t, err)

			t.Run("then manager is cleared in less then 5 minutes", func(t *testing.T) {
				err := wait.Contrail{
					Namespace:     namespace,
					Timeout:       5 * time.Minute,
					RetryInterval: retryInterval,
					Client:        f.Client,
				}.ForManagerDeletion(cluster.Name)
				require.NoError(t, err)
			})
		})
	})
}
