package upgrade

import (
	"context"

	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	upgradeOnChange            = "upgrade.onChange"
	upgradeOnDelete            = "upgrade.onRemove"
	upgradePlansOnChange       = "upgrade.plansOnChange"
	upgradeDeploymentsOnChange = "upgrade.deploymentsOnChange"
	upgradeJobsOnChange        = "upgrade.jobsOnChange"
	upgradeAddonsOnChange      = "upgrade.addonsOnChange"
	upgradeSyncerOnChange      = "upgrade.syncerOnChange"
)

func Register(ctx context.Context, mgmt *config.Management) error {
	upgrades := mgmt.MgmtFactory.Management().V1().Upgrade()
	deployments := mgmt.AppsFactory.Apps().V1().Deployment()
	plans := mgmt.UpgradeFactory.Upgrade().V1().Plan()
	jobs := mgmt.BatchFactory.Batch().V1().Job()
	helms := mgmt.HelmFactory.Helm().V1().HelmChart()
	svc := mgmt.CoreFactory.Core().V1().Service()
	nodes := mgmt.CoreFactory.Core().V1().Node()
	addons := mgmt.MgmtFactory.Management().V1().ManagedAddon()
	settings := mgmt.MgmtFactory.Management().V1().Setting()

	comHandler := &commonHandler{
		upgradeClient: upgrades,
		upgradeCache:  upgrades.Cache(),
	}

	upgradeHandler := &upgradeHandler{
		upgradeClient:    upgrades,
		upgradeCache:     upgrades.Cache(),
		helmChartClient:  helms,
		helmChartCache:   helms.Cache(),
		planClient:       plans,
		planCache:        plans.Cache(),
		deploymentClient: deployments,
		deploymentCache:  deployments.Cache(),
		svcClient:        svc,
		svcCache:         svc.Cache(),
		discovery:        mgmt.ClientSet.Discovery(),
		addonCache:       addons.Cache(),
		commonHandler:    comHandler,
	}

	upgrades.OnChange(ctx, upgradeOnChange, upgradeHandler.onChange)
	upgrades.OnRemove(ctx, upgradeOnDelete, upgradeHandler.onDelete)

	deploymentHandler := &deploymentHandler{
		releaseName:     mgmt.ReleaseName,
		deploymentCache: deployments.Cache(),
		upgradeClient:   upgrades,
		upgradeCache:    upgrades.Cache(),
		commonHandler:   comHandler,
	}

	deployments.OnChange(ctx, upgradeDeploymentsOnChange, deploymentHandler.watchDeployment)

	planHandler := &planHandler{
		upgradeClient: upgrades,
		upgradeCache:  upgrades.Cache(),
		planClient:    plans,
		nodeCache:     nodes.Cache(),
		commonHandler: comHandler,
	}
	plans.OnChange(ctx, upgradePlansOnChange, planHandler.watchUpgradePlans)

	jobHandler := &jobHandler{
		upgradeClient: upgrades,
		upgradeCache:  upgrades.Cache(),
		commonHandler: comHandler,
	}
	jobs.OnChange(ctx, upgradeJobsOnChange, jobHandler.watchUpgradeJobs)

	addonHandler := addonHandler{
		upgradeClient: upgrades,
		addonClient:   addons,
		addonCache:    addons.Cache(),
		commonHandler: comHandler,
	}

	addons.OnChange(ctx, upgradeAddonsOnChange, addonHandler.onAddonChange)

	versionSyncer := newVersionSyncer(mgmt)
	go versionSyncer.start()

	settingHandler := &settingHandler{
		versionSyncer: versionSyncer,
	}
	settings.OnChange(ctx, upgradeSyncerOnChange, settingHandler.syncerOnChange)

	return nil
}
