package mock

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cybozu-go/sabakan/v2"
)

func (d *driver) machineRegister(ctx context.Context, machines []*sabakan.Machine) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, m := range machines {
		if _, ok := d.machines[m.Spec.Serial]; ok {
			return sabakan.ErrConflicted
		}
	}
	for _, m := range machines {
		d.machines[m.Spec.Serial] = m
	}
	return nil
}

func (d *driver) machineGet(ctx context.Context, serial string) (*sabakan.Machine, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	m, ok := d.machines[serial]
	if !ok {
		return nil, sabakan.ErrNotFound
	}

	return m, nil
}

func (d *driver) machineSetState(ctx context.Context, serial string, state sabakan.MachineState) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	m, ok := d.machines[serial]
	if !ok {
		return sabakan.ErrNotFound
	}

	if state == sabakan.StateRetired {
		prefix := serial + "/"
		for k := range d.storage {
			if strings.HasPrefix(k, prefix) {
				return sabakan.ErrBadRequest
			}
		}
	}
	return m.SetState(state)
}

func (d *driver) machinePutLabel(ctx context.Context, serial string, label, value string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	m, ok := d.machines[serial]
	if !ok {
		return sabakan.ErrNotFound
	}
	m.PutLabel(label, value)
	return nil
}

func (d *driver) machineDeleteLabel(ctx context.Context, serial string, label string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	m, ok := d.machines[serial]
	if !ok {
		return sabakan.ErrNotFound
	}
	return m.DeleteLabel(label)
}

func (d *driver) machineSetRetireDate(ctx context.Context, serial string, date time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	m, ok := d.machines[serial]
	if !ok {
		return sabakan.ErrNotFound
	}
	m.Spec.RetireDate = date
	return nil
}

func (d *driver) machineQuery(ctx context.Context, q sabakan.Query) ([]*sabakan.Machine, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	res := make([]*sabakan.Machine, 0)
	for _, m := range d.machines {
		if q.Match(m) {
			res = append(res, m)
		}
	}
	return res, nil
}

func (d *driver) machineDelete(ctx context.Context, serial string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	m, ok := d.machines[serial]
	if !ok {
		return sabakan.ErrNotFound
	}

	if m.Status.State != sabakan.StateRetired {
		return errors.New("non-retired machine cannot be deleted")
	}

	delete(d.machines, serial)
	return nil
}

type machineDriver struct {
	*driver
}

func (d machineDriver) Register(ctx context.Context, machines []*sabakan.Machine) error {
	return d.machineRegister(ctx, machines)
}

func (d machineDriver) Get(ctx context.Context, serial string) (*sabakan.Machine, error) {
	return d.machineGet(ctx, serial)
}

func (d machineDriver) SetState(ctx context.Context, serial string, state sabakan.MachineState) error {
	return d.machineSetState(ctx, serial, state)
}

func (d machineDriver) PutLabel(ctx context.Context, serial string, label, value string) error {
	return d.machinePutLabel(ctx, serial, label, value)
}

func (d machineDriver) DeleteLabel(ctx context.Context, serial string, label string) error {
	return d.machineDeleteLabel(ctx, serial, label)
}

func (d machineDriver) SetRetireDate(ctx context.Context, serial string, date time.Time) error {
	return d.machineSetRetireDate(ctx, serial, date)
}

func (d machineDriver) Query(ctx context.Context, query sabakan.Query) ([]*sabakan.Machine, error) {
	return d.machineQuery(ctx, query)
}

func (d machineDriver) Delete(ctx context.Context, serial string) error {
	return d.machineDelete(ctx, serial)
}
