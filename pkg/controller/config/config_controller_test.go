package config

import (
	"context"
	contrail "github.com/Juniper/contrail-operator/pkg/apis/contrail/v1alpha1"
	"github.com/Juniper/contrail-operator/pkg/volumeclaims"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

type TestCase struct {
	name           string
	initObjs       []runtime.Object
	expectedStatus contrail.ConfigStatus
	fails          bool
	requeued       bool
}

func TestConfig(t *testing.T) {
	scheme, err := contrail.SchemeBuilder.Build()
	require.NoError(t, err, "Failed to build scheme")
	require.NoError(t, core.SchemeBuilder.AddToScheme(scheme), "Failed core.SchemeBuilder.AddToScheme()")
	require.NoError(t, apps.SchemeBuilder.AddToScheme(scheme), "Failed apps.SchemeBuilder.AddToScheme()")

	tests := []*TestCase{
		testcase1(),
		testcase2(),
		testcase3(),
		testcase4(),
		testcase5(),
		testcase6(),
		testcase7(),
		testcase8(),
		testcase9(),
		testcase10(),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := fake.NewFakeClientWithScheme(scheme, tt.initObjs...)

			claims := volumeclaims.New(cl, scheme)
			r := &ReconcileConfig{Client: cl, Scheme: scheme, claims: claims}

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "config-instance",
					Namespace: "default",
				},
			}
			res, err := r.Reconcile(req)

			// check for success or failure
			if tt.fails {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			require.Equal(t, tt.requeued, res.Requeue, "Requeue flag not as expected")
			conf := &contrail.Config{}
			err = cl.Get(context.Background(), req.NamespacedName, conf)
			require.NoError(t, err, "Failed to get status")
			compareConfigStatus(t, tt.expectedStatus, conf.Status)
		})
	}
}

func newConfigInst() *contrail.Config {
	trueVal := true
	falseVal := false
	replica := int32(1)
	return &contrail.Config{
		ObjectMeta: meta.ObjectMeta{
			Name:      "config-instance",
			Namespace: "default",
			Labels:    map[string]string{"contrail_cluster": "config1"},
			OwnerReferences: []meta.OwnerReference{
				{
					Name:       "config1",
					Kind:       "Manager",
					Controller: &trueVal,
				},
			},
		},
		Spec: contrail.ConfigSpec{
			CommonConfiguration: contrail.CommonConfiguration{
				Activate:     &trueVal,
				Create:       &trueVal,
				HostNetwork:  &trueVal,
				Replicas:     &replica,
				NodeSelector: map[string]string{"node-role.kubernetes.io/master": ""},
			},
			ServiceConfiguration: contrail.ConfigConfiguration{
				CassandraInstance:  "cassandra-instance",
				ZookeeperInstance:  "zookeeper-instance",
				KeystoneSecretName: "keystone-adminpass-secret",
				Containers: map[string]*contrail.Container{
					"nodemanagerconfig":    &contrail.Container{Image: "contrail-nodemanager-config"},
					"nodemanageranalytics": &contrail.Container{Image: "contrail-nodemanager-analytics"},
					"config":               &contrail.Container{Image: "contrail-config-api"},
					"analyticsapi":         &contrail.Container{Image: "contrail-analytics-api"},
					"api":                  &contrail.Container{Image: "contrail-controller-config-api"},
					"collector":            &contrail.Container{Image: "contrail-analytics-collector"},
					"devicemanager":        &contrail.Container{Image: "contrail-controller-config-devicemgr"},
					"dnsmasq":              &contrail.Container{Image: "contrail-controller-config-dnsmasq"},
					"init":                 &contrail.Container{Image: "python:alpine"},
					"init2":                &contrail.Container{Image: "busybox"},
					"nodeinit":             &contrail.Container{Image: "contrail-node-init"},
					"redis":                &contrail.Container{Image: "redis"},
					"schematransformer":    &contrail.Container{Image: "contrail-controller-config-schema"},
					"servicemonitor":       &contrail.Container{Image: "contrail-controller-config-svcmonitor"},
					"queryengine":          &contrail.Container{Image: "contrail-analytics-query-engine"},
				},
			},
		},
		Status: contrail.ConfigStatus{Active: &falseVal},
	}
}

func newCassandra() *contrail.Cassandra {
	trueVal := true
	return &contrail.Cassandra{
		ObjectMeta: meta.ObjectMeta{
			Name:      "cassandra-instance",
			Namespace: "default",
		},
		Status: contrail.CassandraStatus{Active: &trueVal},
	}
}

func newRabbitmq() *contrail.Rabbitmq {
	trueVal := true
	return &contrail.Rabbitmq{
		ObjectMeta: meta.ObjectMeta{
			Name:      "rabbitmq-instance",
			Namespace: "default",
			Labels:    map[string]string{"contrail_cluster": "config1"},
		},
		Status: contrail.RabbitmqStatus{Active: &trueVal},
	}
}

func newManager(cfg *contrail.Config) *contrail.Manager {
	return &contrail.Manager{
		ObjectMeta: meta.ObjectMeta{
			Name:      "config1",
			Namespace: "default",
			Labels:    map[string]string{"contrail_cluster": "config1"},
		},
		Spec: contrail.ManagerSpec{
			Services: contrail.Services{
				Config: cfg,
			},
		},
		Status: contrail.ManagerStatus{},
	}
}

func newZookeeper() *contrail.Zookeeper {
	trueVal := true
	replica := int32(1)
	return &contrail.Zookeeper{
		ObjectMeta: meta.ObjectMeta{
			Name:      "zookeeper-instance",
			Namespace: "default",
			Labels:    map[string]string{"contrail_cluster": "config1"},
			OwnerReferences: []meta.OwnerReference{
				{
					Name:       "config1",
					Kind:       "Manager",
					Controller: &trueVal,
				},
			},
		},
		Spec: contrail.ZookeeperSpec{
			CommonConfiguration: contrail.CommonConfiguration{
				Activate:     &trueVal,
				Create:       &trueVal,
				HostNetwork:  &trueVal,
				Replicas:     &replica,
				NodeSelector: map[string]string{"node-role.kubernetes.io/master": ""},
			},
			ServiceConfiguration: contrail.ZookeeperConfiguration{
				Containers: map[string]*contrail.Container{
					"zookeeper":         &contrail.Container{Image: "contrail-controller-zookeeper"},
					"analyticsapi":      &contrail.Container{Image: "contrail-analytics-api"},
					"api":               &contrail.Container{Image: "contrail-controller-config-api"},
					"collector":         &contrail.Container{Image: "contrail-analytics-collector"},
					"devicemanager":     &contrail.Container{Image: "contrail-controller-config-devicemgr"},
					"dnsmasq":           &contrail.Container{Image: "contrail-controller-config-dnsmasq"},
					"init":              &contrail.Container{Image: "python:alpine"},
					"init2":             &contrail.Container{Image: "busybox"},
					"nodeinit":          &contrail.Container{Image: "contrail-node-init"},
					"redis":             &contrail.Container{Image: "redis"},
					"schematransformer": &contrail.Container{Image: "contrail-controller-config-schema"},
					"servicemonitor":    &contrail.Container{Image: "contrail-controller-config-svcmonitor"},
					"queryengine":       &contrail.Container{Image: "contrail-analytics-query-engine"},
				},
			},
		},
		Status: contrail.ZookeeperStatus{Active: &trueVal},
	}
}

func compareConfigStatus(t *testing.T, expectedStatus, realStatus contrail.ConfigStatus) {
	require.NotNil(t, expectedStatus.Active, "expectedStatus.Active should not be nil")
	require.NotNil(t, realStatus.Active, "realStatus.Active Should not be nil")
	assert.Equal(t, *expectedStatus.Active, *realStatus.Active)
}

// ------------------------ TEST CASES ------------------------------------

func testcase1() *TestCase {
	falseVal := false
	cfg := newConfigInst()
	tc := &TestCase{
		name: "create a new statefulset",
		initObjs: []runtime.Object{
			newManager(cfg),
			newZookeeper(),
			newCassandra(),
			newRabbitmq(),
			cfg,
		},
		expectedStatus: contrail.ConfigStatus{Active: &falseVal},
	}
	return tc
}

func testcase2() *TestCase {
	falseVal := false
	cfg := newConfigInst()

	dt := meta.Now()
	cfg.ObjectMeta.DeletionTimestamp = &dt

	tc := &TestCase{
		name: "Config deletion timestamp set",
		initObjs: []runtime.Object{
			newManager(cfg),
			newZookeeper(),
			newCassandra(),
			newRabbitmq(),
			cfg,
		},
		expectedStatus: contrail.ConfigStatus{Active: &falseVal},
	}
	return tc
}

func testcase3() *TestCase {
	falseVal := false
	cfg := newConfigInst()

	command := []string{"bash", "/dummy/run.sh"}
	cfg.Spec.ServiceConfiguration.Containers["config"].Command = command

	cfg.Spec.ServiceConfiguration.Containers["api"].Command = command
	cfg.Spec.ServiceConfiguration.Containers["devicemanager"].Command = command
	cfg.Spec.ServiceConfiguration.Containers["dnsmasq"].Command = command
	cfg.Spec.ServiceConfiguration.Containers["servicemonitor"].Command = command
	cfg.Spec.ServiceConfiguration.Containers["schematransformer"].Command = command
	cfg.Spec.ServiceConfiguration.Containers["analyticsapi"].Command = command
	cfg.Spec.ServiceConfiguration.Containers["collector"].Command = command
	cfg.Spec.ServiceConfiguration.Containers["redis"].Command = command
	cfg.Spec.ServiceConfiguration.Containers["nodemanagerconfig"].Command = command
	cfg.Spec.ServiceConfiguration.Containers["nodemanageranalytics"].Command = command

	tc := &TestCase{
		name: "Preset start command for containers",
		initObjs: []runtime.Object{
			newManager(cfg),
			newZookeeper(),
			newCassandra(),
			newRabbitmq(),
			cfg,
		},
		expectedStatus: contrail.ConfigStatus{Active: &falseVal},
	}
	return tc
}

func testcase4() *TestCase {
	falseVal := false
	cfg := newConfigInst()
	zkp := newZookeeper()

	zkp.Status.Active = &falseVal

	tc := &TestCase{
		name: "Config service not up",
		initObjs: []runtime.Object{
			newManager(cfg),
			zkp,
			newCassandra(),
			newRabbitmq(),
			cfg,
		},
		expectedStatus: contrail.ConfigStatus{Active: &falseVal},
	}
	return tc
}

func testcase5() *TestCase {
	falseVal := false
	cfg := newConfigInst()

	cfg.Spec.ServiceConfiguration.Storage.Path = "my-storage-path"
	cfg.Spec.ServiceConfiguration.Storage.Size = "1G"
	cfg.Spec.CommonConfiguration.NodeSelector = map[string]string{
		"selector1": "1",
	}
	tc := &TestCase{
		name: "Set Storage Info",
		initObjs: []runtime.Object{
			newManager(cfg),
			newZookeeper(),
			newCassandra(),
			newRabbitmq(),
			cfg,
		},
		expectedStatus: contrail.ConfigStatus{Active: &falseVal},
	}
	return tc
}

func testcase6() *TestCase {
	falseVal := false
	cfg := newConfigInst()

	tc := &TestCase{
		name: "No Manager Instance Found",
		initObjs: []runtime.Object{
			// newManager(cfg),
			newZookeeper(),
			newCassandra(),
			newRabbitmq(),
			cfg,
		},
		expectedStatus: contrail.ConfigStatus{Active: &falseVal},
		fails:          true,
	}
	return tc
}

func testcase7() *TestCase {
	falseVal := false
	cfg := newConfigInst()

	cfg.Spec.ServiceConfiguration.NodeManager = &falseVal

	tc := &TestCase{
		name: "Object is not a node manager",
		initObjs: []runtime.Object{
			newManager(cfg),
			newZookeeper(),
			newCassandra(),
			newRabbitmq(),
			cfg,
		},
		expectedStatus: contrail.ConfigStatus{Active: &falseVal},
	}
	return tc
}

func testcase8() *TestCase {
	falseVal := false
	cfg := newConfigInst()

	command := []string{"bash", "/runner/run.sh"}
	cfg.Spec.ServiceConfiguration.Containers["config"].Command = command
	_, found := cfg.Spec.ServiceConfiguration.Containers["nodemanagerconfig"]
	if found {
		delete(cfg.Spec.ServiceConfiguration.Containers, "nodemanagerconfig")
	}

	_, found = cfg.Spec.ServiceConfiguration.Containers["nodemanageranalytics"]
	if found {
		delete(cfg.Spec.ServiceConfiguration.Containers, "nodemanageranalytics")
	}

	tc := &TestCase{
		name: "Remove Node Manager templates if Node Manager containers not listed",
		initObjs: []runtime.Object{
			newManager(cfg),
			newZookeeper(),
			newCassandra(),
			newRabbitmq(),
			cfg,
		},
		expectedStatus: contrail.ConfigStatus{Active: &falseVal},
	}
	return tc
}

func testcase9() *TestCase {
	falseVal := false
	cfg := newConfigInst()

	command := []string{"bash", "/runner/run.sh"}
	cfg.Spec.ServiceConfiguration.Containers["init"].Command = command

	tc := &TestCase{
		name: "Preset Init command",
		initObjs: []runtime.Object{
			newManager(cfg),
			newZookeeper(),
			newCassandra(),
			newRabbitmq(),
			cfg,
		},
		expectedStatus: contrail.ConfigStatus{Active: &falseVal},
	}
	return tc
}

func testcase10() *TestCase {
	trueVal := true
	falseVal := false
	cfg := newConfigInst()

	cfg.Status.Active = &trueVal
	cfg.Status.ConfigChanged = &trueVal

	tc := &TestCase{
		name: "Indicate that config changed",
		initObjs: []runtime.Object{
			newManager(cfg),
			newZookeeper(),
			newCassandra(),
			newRabbitmq(),
			cfg,
		},
		requeued:       trueVal,
		expectedStatus: contrail.ConfigStatus{Active: &falseVal},
	}
	return tc
}