package mlstate

import (
	"fmt"
	"pfi/sensorbee/pystate/py"
	"pfi/sensorbee/sensorbee/core"
	"pfi/sensorbee/sensorbee/data"
)

type PyMLState struct {
	mdl py.ObjectModule
	ins py.ObjectModule

	bucket    data.Array
	batchSize int
}

func NewPyMLState(modulePathName, moduleName, className string, batchSize int) (*PyMLState, error) {
	py.ImportSysAndAppendPath(modulePathName)

	mdl, err := py.LoadModule(moduleName)
	if err != nil {
		return nil, err
	}

	ins, err := mdl.GetInstance(className)
	if err != nil {
		mdl.DecRef()
		return nil, err
	}

	return &PyMLState{
		mdl:       mdl,
		ins:       ins,
		bucket:    make(data.Array, 0, batchSize),
		batchSize: batchSize,
	}, nil
}

func (s *PyMLState) Terminate(ctx *core.Context) error {
	s.ins.DecRef()
	s.mdl.DecRef()
	return nil
}

func (s *PyMLState) Write(ctx *core.Context, t *core.Tuple) error {
	s.bucket = append(s.bucket, t.Data)

	var err error
	if len(s.bucket) >= s.batchSize {
		_, err = s.Fit(ctx, s.bucket)
		s.bucket = s.bucket[:0] // clear slice but keep capacity
	}

	return err
}

// Fit receives data.Array type but it assumes `[]data.Map` type
// for passing arguments to `fit` method.
func (s *PyMLState) Fit(ctx *core.Context, bucket data.Array) (data.Value, error) {
	return s.ins.Call("fit", bucket)
}

func (s *PyMLState) FitMap(ctx *core.Context, bucket []data.Map) (data.Value, error) {
	args := make(data.Array, len(bucket))
	for i, v := range bucket {
		args[i] = v
	}
	return s.ins.Call("fit", args)
}

func PyMLFit(ctx *core.Context, stateName string, bucket []data.Map) (data.Value, error) {
	s, err := lookupPyMLState(ctx, stateName)
	if err != nil {
		return nil, err
	}

	return s.FitMap(ctx, bucket)
}

// TODO: PyMLPredict

func lookupPyMLState(ctx *core.Context, stateName string) (*PyMLState, error) {
	st, err := ctx.SharedStates.Get(stateName)
	if err != nil {
		return nil, err
	}

	if s, ok := st.(*PyMLState); ok {
		return s, nil
	}

	return nil, fmt.Errorf("state '%v' isn't a PyMLState", stateName)
}