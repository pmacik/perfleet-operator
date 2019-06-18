package podfleet

import (
	"context"
	"fmt"

	podfleetoperatorv1alpha1 "github.com/pmacik/podfleet-operator/pkg/apis/podfleetoperator/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_podfleet")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new PodFleet Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePodFleet{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("podfleet-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource PodFleet
	err = c.Watch(&source.Kind{Type: &podfleetoperatorv1alpha1.PodFleet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner PodFleet
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &podfleetoperatorv1alpha1.PodFleet{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcilePodFleet implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcilePodFleet{}

// ReconcilePodFleet reconciles a PodFleet object
type ReconcilePodFleet struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a PodFleet object and makes changes based on the state read
// and what is in the PodFleet.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePodFleet) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	//reqLogger.Info("Reconciling PodFleet")

	// Fetch the PodFleet podFleet
	podFleet := &podfleetoperatorv1alpha1.PodFleet{}
	err := getPodFleet(podFleet, request.NamespacedName, r)
	//err := r.client.Get(context.TODO(), request.NamespacedName, podFleet)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return doNotRequeue()
		}
		// Error reading the object - requeue the request.
		return requeueWithError(err)
	}

	var status = podFleet.Status
	//log.Info("DEBUG", "status", status)
	if !status.Started { // initial status
		log.Info("Init status")
		//log.Info("DEBUG", "status", status)
		status.Started = true
		//log.Info("DEBUG", "status", status)
		status.WarmingUp = true
		//log.Info("DEBUG", "status", status)
		err = updateStatus(podFleet, status, r)
		//log.Info("DEBUG", "status", status)
		if err != nil {
			return requeueWithError(err)
		}
		return requeue()
	}

	// List all pods owned by this PodFleet instance
	podList := &corev1.PodList{}
	lbs := map[string]string{
		"app":     podFleet.Name,
		"version": "v0.0.1",
	}
	//reqLogger.Info("(????????) Looking for Pods", "pod.selector", lbs)
	labelSelector := labels.SelectorFromSet(lbs)
	listOpts := &client.ListOptions{
		Namespace:     podFleet.Namespace,
		LabelSelector: labelSelector,
	}
	if err = r.client.List(context.TODO(), listOpts, podList); err != nil {
		return requeueWithError(err)
	}
	// Filter pods
	var pending []corev1.Pod
	var running []corev1.Pod
	var completed []corev1.Pod
	var workers []corev1.Pod

	for _, pod := range podList.Items {
		if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
			continue
		}
		if pod.Status.Phase == corev1.PodRunning {
			running = append(running, pod)
			workers = append(workers, pod)
		} else if pod.Status.Phase == corev1.PodPending {
			pending = append(pending, pod)
			workers = append(workers, pod)
		} else if pod.Status.Phase == corev1.PodSucceeded {
			completed = append(completed, pod)
			workers = append(workers, pod)
		}

	}
	pendingCount := int32(len(pending))
	runningCount := int32(len(running))
	completedCount := int32(len(completed))
	createdCount := pendingCount + runningCount + completedCount

	status.WorkersPending = pendingCount
	status.WorkersWorking = runningCount
	status.WorkersDone = completedCount

	err = updateStatus(podFleet, status, r)
	if err != nil {
		return requeueWithError(err)
	}

	//scale up pods
	if status.WarmingUp && createdCount < podFleet.Spec.Workers && pendingCount == 0 {
		//reqLogger.Info("(++++++++) Scaling up pods...", "Workeres already created", createdCount, "Workers required", podFleet.Spec.Workers)
		// Define a new Pod object
		pod := newPodForCR(podFleet, createdCount)

		// Set PodFleet instance as the owner and controller
		if err := controllerutil.SetControllerReference(podFleet, pod, r.scheme); err != nil {
			return requeueWithError(err)
		}
		//reqLogger.Info("(++++++++) Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			reqLogger.Error(err, "Failed to create pod", "pod.name", pod.Name)
			return requeueWithError(err)
		}
		return requeue()
	}

	if createdCount == podFleet.Spec.Workers {
		status.WarmingUp = false
		err = updateStatus(podFleet, status, r)
		if err != nil {
			return requeueWithError(err)
		}
		//return requeue()
	}

	if status.WarmingUp || runningCount > 0 {
		return requeue()
	}

	//All is done
	log.Info("(!!!!!!!!) Farewell - my work is done.")
	err = deletePodFleet(podFleet, r)
	if err != nil {
		log.Error(err, "Failed to delete the podfleet", "podfleet.name", podFleet.ObjectMeta.Name)
		requeueWithError(err)
	}
	return doNotRequeue()
}

func requeueWithError(err error) (reconcile.Result, error) {
	//log.Info("requeueWithError")
	return reconcile.Result{}, err
}

func requeue() (reconcile.Result, error) {
	//log.Info("requeue")
	return reconcile.Result{Requeue: true}, nil
}

func doNotRequeue() (reconcile.Result, error) {
	//log.Info("doNotRequeue")
	return reconcile.Result{Requeue: false}, nil
}

func deletePodFleet(podFleet *podfleetoperatorv1alpha1.PodFleet, r *ReconcilePodFleet) error {
	//log.Info("deletePodFleet")
	return r.client.Delete(context.TODO(), podFleet)
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *podfleetoperatorv1alpha1.PodFleet, index int32) *corev1.Pod {
	//log.Info("newPodForCR")
	labels := map[string]string{
		"app":     cr.Name,
		"version": "v0.0.1",
		"index":   fmt.Sprintf("%d", index),
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: cr.Name + "-member-",
			Namespace:    cr.Namespace,
			Labels:       labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "15"},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
}

func updateStatus(podFleet *podfleetoperatorv1alpha1.PodFleet, status podfleetoperatorv1alpha1.PodFleetStatus, r *ReconcilePodFleet) error {
	//log.Info("updateStatus")
	podFleet.Status = status
	err := r.client.Status().Update(context.TODO(), podFleet)
	if err != nil {
		log.Error(err, "Failed to update PodFleet status")
		return err
	}
	return nil
}

func getPodFleet(podFleet *podfleetoperatorv1alpha1.PodFleet, namespacedName types.NamespacedName, r *ReconcilePodFleet) error {
	//log.Info("getPodFleet")
	err := r.client.Get(context.TODO(), namespacedName, podFleet)
	return err
}
