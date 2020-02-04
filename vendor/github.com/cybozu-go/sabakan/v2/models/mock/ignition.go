package mock

import (
	"context"
	"sort"
	"sync"

	"github.com/cybozu-go/sabakan/v2"
	version "github.com/hashicorp/go-version"
)

type ignitionDriver struct {
	mu        sync.Mutex
	ignitions map[string]map[string]*sabakan.IgnitionTemplate
}

func newIgnitionDriver() *ignitionDriver {
	return &ignitionDriver{
		ignitions: make(map[string]map[string]*sabakan.IgnitionTemplate),
	}
}

func (d *ignitionDriver) PutTemplate(ctx context.Context, role, id string, tmpl *sabakan.IgnitionTemplate) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	templateMap := d.ignitions[role]
	if templateMap == nil {
		templateMap = make(map[string]*sabakan.IgnitionTemplate)
		d.ignitions[role] = templateMap
	}
	if _, ok := templateMap[id]; ok {
		return sabakan.ErrConflicted
	}
	templateMap[id] = tmpl
	return nil
}

func (d *ignitionDriver) GetTemplateIDs(ctx context.Context, role string) ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	templateMap, ok := d.ignitions[role]
	if !ok {
		return nil, nil
	}

	versions := make([]*version.Version, 0, len(templateMap))
	for k := range templateMap {
		ver, err := version.NewVersion(k)
		if err != nil {
			return nil, err
		}
		versions = append(versions, ver)
	}

	sort.Sort(version.Collection(versions))

	result := make([]string, len(versions))
	for i, ver := range versions {
		result[i] = ver.Original()
	}
	return result, nil
}

func (d *ignitionDriver) GetTemplate(ctx context.Context, role string, id string) (*sabakan.IgnitionTemplate, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	res, ok := d.ignitions[role][id]
	if !ok {
		return nil, sabakan.ErrNotFound
	}
	return res, nil
}

func (d *ignitionDriver) DeleteTemplate(ctx context.Context, role string, id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	ids, ok := d.ignitions[role]
	if !ok {
		return sabakan.ErrNotFound
	}
	if _, ok := ids[id]; !ok {
		return sabakan.ErrNotFound
	}
	delete(ids, id)
	return nil
}
