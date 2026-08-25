package ray_tracing

import "testing"

type countingPreparedState struct{}

func (*countingPreparedState) preparedIntegratorState() {}

type countingSplatKernel struct {
	prepareCalls int
}

func (k *countingSplatKernel) Prepare(*RenderJob) (PreparedIntegratorState, error) {
	k.prepareCalls++
	return &countingPreparedState{}, nil
}

func (*countingSplatKernel) WorkCount(*RenderJob, PreparedIntegratorState) int64 {
	return 0
}

func (*countingSplatKernel) TraceSample(*RenderJob, PreparedIntegratorState, int64) []FilmSplat {
	return nil
}

func TestSplatIntegratorConsumesSinglePreparedState(t *testing.T) {
	kernel := &countingSplatKernel{}
	integrator := &splatSceneIntegrator{kernel: kernel}
	job := &RenderJob{}
	prepared, err := integrator.Prepare(job)
	if err != nil {
		t.Fatal(err)
	}
	if err := integrator.Run(job, prepared); err != nil {
		t.Fatal(err)
	}
	if kernel.prepareCalls != 1 {
		t.Fatalf("Prepare called %d times, want once", kernel.prepareCalls)
	}
}
