package e2e

import (
	"fmt"
	"strings"
	"testing"

	"github.com/argoproj/gitops-engine/pkg/health"
	. "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/stretchr/testify/require"

	. "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v3/test/e2e/fixture"
	. "github.com/argoproj/argo-cd/v3/test/e2e/fixture/app"
	"github.com/argoproj/argo-cd/v3/util/errors"
	"github.com/argoproj/argo-cd/v3/util/rand"
)

// when you selectively sync, only selected resources should be synced, but the app will be out of sync
func TestSelectiveSync(t *testing.T) {
	Given(t).
		Path("guestbook").
		SelectedResource(":Service:guestbook-ui").
		When().
		CreateApp().
		Sync().
		Then().
		Expect(Success("")).
		Expect(OperationPhaseIs(OperationSucceeded)).
		Expect(SyncStatusIs(SyncStatusCodeOutOfSync)).
		Expect(ResourceHealthIs("Service", "guestbook-ui", health.HealthStatusHealthy)).
		Expect(ResourceHealthIs("Deployment", "guestbook-ui", health.HealthStatusMissing))
}

// when running selective sync, hooks do not run
// hooks don't run even if all resources are selected
func TestSelectiveSyncDoesNotRunHooks(t *testing.T) {
	Given(t).
		Path("hook").
		SelectedResource(":Pod:pod").
		When().
		CreateApp().
		Sync().
		Then().
		Expect(Success("")).
		Expect(OperationPhaseIs(OperationSucceeded)).
		Expect(SyncStatusIs(SyncStatusCodeSynced)).
		Expect(HealthIs(health.HealthStatusHealthy)).
		Expect(ResourceHealthIs("Pod", "pod", health.HealthStatusHealthy)).
		Expect(ResourceResultNumbering(1))
}

func TestSelectiveSyncWithoutNamespace(t *testing.T) {
	selectedResourceNamespace := getNewNamespace(t)
	defer func() {
		if !t.Skipped() {
			errors.NewHandler(t).FailOnErr(fixture.Run("", "kubectl", "delete", "namespace", selectedResourceNamespace))
		}
	}()
	Given(t).
		Prune(true).
		Path("guestbook-with-namespace").
		And(func() {
			errors.NewHandler(t).FailOnErr(fixture.Run("", "kubectl", "create", "namespace", selectedResourceNamespace))
		}).
		SelectedResource("apps:Deployment:guestbook-ui").
		When().
		PatchFile("guestbook-ui-deployment-ns.yaml", fmt.Sprintf(`[{"op": "replace", "path": "/metadata/namespace", "value": %q}]`, selectedResourceNamespace)).
		PatchFile("guestbook-ui-svc-ns.yaml", fmt.Sprintf(`[{"op": "replace", "path": "/metadata/namespace", "value": %q}]`, selectedResourceNamespace)).
		CreateApp().
		Sync().
		Then().
		Expect(Success("")).
		Expect(OperationPhaseIs(OperationSucceeded)).
		Expect(SyncStatusIs(SyncStatusCodeOutOfSync)).
		Expect(ResourceHealthWithNamespaceIs("Deployment", "guestbook-ui", selectedResourceNamespace, health.HealthStatusHealthy)).
		Expect(ResourceHealthWithNamespaceIs("Deployment", "guestbook-ui", fixture.DeploymentNamespace(), health.HealthStatusHealthy)).
		Expect(ResourceSyncStatusWithNamespaceIs("Deployment", "guestbook-ui", selectedResourceNamespace, SyncStatusCodeSynced)).
		Expect(ResourceSyncStatusWithNamespaceIs("Deployment", "guestbook-ui", fixture.DeploymentNamespace(), SyncStatusCodeSynced))
}

// In selectedResource to sync, namespace is provided
func TestSelectiveSyncWithNamespace(t *testing.T) {
	selectedResourceNamespace := getNewNamespace(t)
	defer func() {
		if !t.Skipped() {
			errors.NewHandler(t).FailOnErr(fixture.Run("", "kubectl", "delete", "namespace", selectedResourceNamespace))
		}
	}()
	Given(t).
		Prune(true).
		Path("guestbook-with-namespace").
		And(func() {
			errors.NewHandler(t).FailOnErr(fixture.Run("", "kubectl", "create", "namespace", selectedResourceNamespace))
		}).
		SelectedResource(fmt.Sprintf("apps:Deployment:%s/guestbook-ui", selectedResourceNamespace)).
		When().
		PatchFile("guestbook-ui-deployment-ns.yaml", fmt.Sprintf(`[{"op": "replace", "path": "/metadata/namespace", "value": %q}]`, selectedResourceNamespace)).
		PatchFile("guestbook-ui-svc-ns.yaml", fmt.Sprintf(`[{"op": "replace", "path": "/metadata/namespace", "value": %q}]`, selectedResourceNamespace)).
		CreateApp().
		Sync().
		Then().
		Expect(Success("")).
		Expect(OperationPhaseIs(OperationSucceeded)).
		Expect(SyncStatusIs(SyncStatusCodeOutOfSync)).
		Expect(ResourceHealthWithNamespaceIs("Deployment", "guestbook-ui", selectedResourceNamespace, health.HealthStatusHealthy)).
		Expect(ResourceHealthWithNamespaceIs("Deployment", "guestbook-ui", fixture.DeploymentNamespace(), health.HealthStatusMissing)).
		Expect(ResourceSyncStatusWithNamespaceIs("Deployment", "guestbook-ui", selectedResourceNamespace, SyncStatusCodeSynced)).
		Expect(ResourceSyncStatusWithNamespaceIs("Deployment", "guestbook-ui", fixture.DeploymentNamespace(), SyncStatusCodeOutOfSync))
}

func getNewNamespace(t *testing.T) string {
	t.Helper()
	randStr, err := rand.String(5)
	require.NoError(t, err)
	postFix := "-" + strings.ToLower(randStr)
	name := fixture.DnsFriendly(t.Name(), "")
	return fixture.DnsFriendly("argocd-e2e-"+name, postFix)
}
