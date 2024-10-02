package controllers

import (
	"context"

	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api-provider-azure/azure/scope"
	"sigs.k8s.io/cluster-api-provider-azure/azure/services/resourceskus"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/cluster-api/util/annotations"

	"github.com/pkg/errors"
	"k8s.io/client-go/tools/record"
	infrav1 "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-azure/pkg/coalescing"
	"sigs.k8s.io/cluster-api-provider-azure/util/reconciler"
	"sigs.k8s.io/cluster-api-provider-azure/util/tele"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

type AzureMachineTemplateReconciler struct {
	client.Client
	Recorder         record.EventRecorder
	Timeouts         reconciler.Timeouts
	WatchFilterValue string
}

func NewAzureMachineTemplateReconciler(client client.Client, recorder record.EventRecorder, timeouts reconciler.Timeouts, watchFilterValue string) *AzureMachineTemplateReconciler {
	return &AzureMachineTemplateReconciler{
		Client:           client,
		Recorder:         recorder,
		Timeouts:         timeouts,
		WatchFilterValue: watchFilterValue,
	}
}

func (amtr *AzureMachineTemplateReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options Options) error {
	ctx, log, done := tele.StartSpanWithLogger(ctx,
		"controllers.AzureMachineTemplateReconciler.SetupWithManager",
		tele.KVP("controller", "AzureMachineTemplate"),
	)
	defer done()

	var r reconcile.Reconciler = amtr
	coalescing.NewReconciler(amtr, options.Cache, log)
	if options.Cache != nil {
		r = coalescing.NewReconciler(amtr, options.Cache, log)
	}

	azureMachineTemplateMapper, err := util.ClusterToTypedObjectsMapper(amtr.Client, &infrav1.AzureMachineTemplateList{}, mgr.GetScheme())
	if err != nil {
		return errors.Wrap(err, "failed to create mapper for Cluster to AzureMachineTemplates")
	}

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options.Options).
		For(&infrav1.AzureMachineTemplate{}).
		WithEventFilter(predicates.ResourceHasFilterLabel(log, amtr.WatchFilterValue)).
		// Add a watch on Clusters to requeue when the infraRef is set. This is needed because the infraRef is not initially
		// set in Clusters created from a ClusterClass.
		Watches(
			&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(azureMachineTemplateMapper),
			builder.WithPredicates(
				predicates.ClusterUnpausedAndInfrastructureReady(log),
				predicates.ResourceNotPausedAndHasFilterLabel(log, amtr.WatchFilterValue),
			),
		).
		Complete(r)
}

func (amtr *AzureMachineTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx, cancel := context.WithTimeout(ctx, amtr.Timeouts.DefaultedLoopTimeout())
	defer cancel()

	ctx, log, done := tele.StartSpanWithLogger(ctx, "controllers.AzureMachineTemplateReconciler.Reconcile",
		tele.KVP("namespace", req.Namespace),
		tele.KVP("name", req.Name),
		tele.KVP("kind", "AzureMachineTemplate"),
	)
	defer done()

	// Fetch the AzureMachineTemplate instance
	azureMachineTemplate := &infrav1.AzureMachineTemplate{}
	err := amtr.Get(ctx, req.NamespacedName, azureMachineTemplate)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("object was not found")
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !azureMachineTemplate.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, amtr.Client, azureMachineTemplate.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if cluster == nil {
		log.Info("Cluster Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("cluster", cluster.Name)

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, azureMachineTemplate) {
		log.Info("AzureMachineTemplate or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	// only look at azure clusters
	if cluster.Spec.InfrastructureRef == nil {
		log.Info("infra ref is nil")
		return ctrl.Result{}, nil
	}
	if cluster.Spec.InfrastructureRef.Kind != infrav1.AzureClusterKind {
		log.WithValues("kind", cluster.Spec.InfrastructureRef.Kind).Info("infra ref was not an AzureCluster")
		return ctrl.Result{}, nil
	}

	// fetch the corresponding azure cluster
	azureCluster := &infrav1.AzureCluster{}
	azureClusterName := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	if err := amtr.Get(ctx, azureClusterName, azureCluster); err != nil {
		log.Error(err, "failed to fetch AzureCluster")
		return ctrl.Result{}, err
	}

	// Create the scope.
	clusterScope, err := scope.NewClusterScope(ctx, scope.ClusterScopeParams{
		Client:       amtr.Client,
		Cluster:      cluster,
		AzureCluster: azureCluster,
		Timeouts:     amtr.Timeouts,
	})
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to create scope")
	}

	if azureMachineTemplate.Status.Capacity != nil {
		log.V(4).Info("capacity already set, done reconciling")
		return ctrl.Result{}, nil
	}

	helper, err := patch.NewHelper(azureMachineTemplate, amtr.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to init patch helper")
	}

	defer func() {
		if err := helper.Patch(ctx, azureMachineTemplate); err != nil {
			reterr = err
		}
	}()

	skuCache, err := resourceskus.GetCache(clusterScope, clusterScope.Location())
	if err != nil {
		return ctrl.Result{}, err
	}

	vmSKU, err := skuCache.Get(ctx, azureMachineTemplate.Spec.Template.Spec.VMSize, resourceskus.VirtualMachines)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to get VM SKU %s in compute api", azureMachineTemplate.Spec.Template.Spec.VMSize)
	}

	azureMachineTemplate.Status.Capacity = resourceskus.MapCapabilitiesToResourceList(vmSKU.Capabilities)

	return ctrl.Result{}, nil
}
