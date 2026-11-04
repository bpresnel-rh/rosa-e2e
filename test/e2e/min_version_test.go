//go:build E2Etests

package e2e

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-online/rosa-e2e/pkg/framework"
	"github.com/openshift-online/rosa-e2e/pkg/labels"
	"github.com/openshift-online/rosa-e2e/pkg/verifiers"
)

var _ = Describe("Minimum OCP Version Coverage", labels.Critical, labels.Positive, labels.MinVersion, func() {
	BeforeEach(func() {
		// Require CLUSTER_ID to be set (cluster is pre-provisioned by openshift/release job)
		if cfg.ClusterID == "" {
			Skip("CLUSTER_ID not set; minimum version tests require a pre-provisioned cluster")
		}
	})

	It("verifies HCP cluster health at minimum OCP version from prod:canary", labels.HCP, func(ctx context.Context) {
		tc := framework.NewTestContext(cfg, conn)

		By("Detecting cluster topology")
		topology := tc.Topology()
		Expect(topology).NotTo(BeEmpty(), "cluster topology detection failed")

		if !tc.IsHCP() {
			Skip("HCP topology required for minimum version coverage")
		}

		By("Verifying cluster is ready in OCM")
		Expect(verifiers.VerifyClusterReadyWithContext(ctx, tc.Connection(), cfg.ClusterID)).To(Succeed())

		By("Verifying cluster belongs to prod:canary sector")
		Expect(framework.VerifyClusterInSector(ctx, tc.Connection(), cfg.ClusterID, "prod:canary")).To(Succeed())

		By("Verifying cluster OCP version is valid")
		version, err := framework.GetClusterVersion(ctx, tc.Connection(), cfg.ClusterID)
		Expect(err).NotTo(HaveOccurred())
		Expect(version).NotTo(BeEmpty(), "cluster OCP version is empty")

		By("Obtaining cluster credentials")
		kubeConfig, err := framework.GetClusterCredentialsWithContext(ctx, tc.Connection(), cfg.ClusterID)
		Expect(err).NotTo(HaveOccurred())

		By("Verifying worker nodes are Ready")
		kubeClient, err := framework.NewKubeClient(kubeConfig)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() error {
			ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()
			return verifiers.RunVerifiers(ctxWithTimeout, kubeClient,
				verifiers.VerifyAllNodesReady(),
			)
		}).WithTimeout(15 * time.Minute).WithPolling(30 * time.Second).Should(Succeed())

		By("Verifying cluster operators are Ready")
		dynClient, err := framework.NewDynamicClient(kubeConfig)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() error {
			ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()
			return verifiers.VerifyClusterOperatorsReady(ctxWithTimeout, dynClient)
		}).WithTimeout(15 * time.Minute).WithPolling(30 * time.Second).Should(Succeed())
	})

})
