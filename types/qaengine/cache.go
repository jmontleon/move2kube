/*
 *  Copyright IBM Corporation 2021
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package qaengine

import (
	"fmt"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/types"
	"github.com/sirupsen/logrus"
)

// QACacheKind defines kind of QA Cache
const QACacheKind types.Kind = "QACache"

// Cache stores the answers for reuse
type Cache struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             CacheSpec `yaml:"spec,omitempty"`
}

// CacheSpec stores the cache data
type CacheSpec struct {
	file             string `yaml:"-"`
	persistPasswords bool   `yaml:"-"`
	// Problems stores the list of problems with resolutions
	Problems []Problem `yaml:"solutions"`
}

// NewCache creates new cache instance
func NewCache(file string, persistPasswords bool) (cache *Cache) {
	return &Cache{
		TypeMeta: types.TypeMeta{
			Kind:       string(QACacheKind),
			APIVersion: types.SchemeGroupVersion.String(),
		},
		Spec: CacheSpec{
			file:             file,
			persistPasswords: persistPasswords,
		},
	}
}

// Load loads and merges cache
func (cache *Cache) Load() error {
	c := Cache{}
	if err := common.ReadMove2KubeYaml(cache.Spec.file, &c); err != nil {
		logrus.Errorf("Unable to load the cache file at path %s Error: %q", cache.Spec.file, err)
		return err
	}
	cache.merge(c)
	return nil
}

// Write writes cache to disk
func (cache *Cache) Write() error {
	err := common.WriteYaml(cache.Spec.file, cache)
	if err != nil {
		logrus.Warnf("Unable to write cache : %s", err)
	}
	return err
}

// AddSolution adds a problem to solution cache
func (cache *Cache) AddSolution(p Problem) error {
	if !cache.Spec.persistPasswords && p.Type == PasswordSolutionFormType {
		err := fmt.Errorf("passwords are not added to the cache")
		logrus.Debug(err)
		return err
	}
	if p.Answer == nil {
		err := fmt.Errorf("unresolved problem. Not going to be added to cache")
		logrus.Warn(err)
		return err
	}
	added := false
	for i, cp := range cache.Spec.Problems {
		if cp.ID == p.ID {
			logrus.Debugf("A solution already exists in cache for [%s], rewriting", p.Desc)
			cache.Spec.Problems[i] = p
			added = true
			break
		}
	}
	if !added {
		cache.Spec.Problems = append(cache.Spec.Problems, p)
	}
	if err := cache.Write(); err != nil {
		logrus.Errorf("Failed to write to the cache file. Error: %q", err)
		return err
	}
	return nil
}

// GetSolution reads a solution for the problem
func (cache *Cache) GetSolution(p Problem) (Problem, error) {
	if p.Answer != nil {
		logrus.Warnf("Problem already solved.")
		return p, nil
	}
	for _, cp := range cache.Spec.Problems {
		if (cp.ID == p.ID || cp.matches(p)) && cp.Answer != nil {
			p.Answer = cp.Answer
			return p, nil
		}
	}
	return p, fmt.Errorf("the problem %+v was not found in the cache", p)
}

func (cache *Cache) merge(c Cache) {
	for _, p := range c.Spec.Problems {
		found := false
		for _, op := range cache.Spec.Problems {
			if op.matches(p) {
				logrus.Warnf("There are two or more answers for %s in cache. Ignoring latter ones.", p.Desc)
				found = true
				break
			}
		}
		if !found {
			cache.Spec.Problems = append(cache.Spec.Problems, p)
		}
	}
}
