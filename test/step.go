// +build test

package test

import (
	"context"
	"fmt"
	. "github.com/argoproj-labs/argo-dataflow/api/v1alpha1"
	sharedutil "github.com/argoproj-labs/argo-dataflow/shared/util"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"log"
	"time"
)

var (
	stepInterface = dynamicInterface.Resource(StepGroupVersionResource).Namespace(namespace)
)

func MessagesPending(s Step) bool {
	return !NothingPending(s)
}

func NothingPending(s Step) bool {
	return s.Status.SourceStatuses.GetPending() == 0
}

func WaitForStep(opts ...interface{}) {

	var (
		listOptions = metav1.ListOptions{}
		f           = func(s Step) bool { return s.Status.Phase == StepRunning }
	)
	for _, o := range opts {
		switch v := o.(type) {
		case string:
			listOptions.FieldSelector = "metadata.name=" + v
		case func(Step) bool:
			f = v
		default:
			panic("un-supported option type")
		}
	}
	log.Printf("waiting for step %q %q\n", sharedutil.MustJSON(listOptions), sharedutil.GetFuncName(f))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	w, err := stepInterface.Watch(ctx, listOptions)
	if err != nil {
		panic(err)
	}
	defer w.Stop()
	for {
		select {
		case <-ctx.Done():
			panic(fmt.Errorf("failed to wait for step: %w", ctx.Err()))
		case e := <-w.ResultChan():
			un, ok := e.Object.(*unstructured.Unstructured)
			if !ok {
				panic(errors.FromObject(e.Object))
			}
			x := StepFromUnstructured(un)
			log.Println(fmt.Sprintf("step %q is %s %q", x.Name, x.Status.Phase, x.Status.Message))
			if f(x) {
				return
			}
		}
	}
}