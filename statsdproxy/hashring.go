// functions to build and maintain a hashring
package statsdproxy

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"log"
	"sort"
)

const (
	MAXRINGSIZE = 50
)

type HashRingID uint32
type HashRing struct {
	Backends []*StatsDBackend
	Mirror   bool
}

// NewHashRing initializes a hash ring
func NewHashRing(mirror bool) *HashRing {
	ret := HashRing{}
	ret.Backends = make([]*StatsDBackend, 0, MAXRINGSIZE)
	ret.Mirror = mirror
	return &ret
}

// Add a new server instance into the hashring
// accepts an instance of StatsDBackend as parameter
// returns the sorted hashring with the newly appended server and error
func (ring *HashRing) Add(backend *StatsDBackend) error {
	if !backend.Alive() {
		errMsg := fmt.Sprintf("Backend %s:%d doesn't seem to be alive.", backend.Host,
			backend.Port)
		return errors.New(errMsg)
	}
	ring.Backends = append(ring.Backends, backend)
	sort.Sort(ByHashRingID(ring.Backends))
	return nil
}

// GetHashRingPosition returns a position in a hashring. The logic is ripped from
// libketama and uses MD5 to determine the position
//
// accepts a string to hash
//
// returns a HashRingID and error
func GetHashRingPosition(data string) (HashRingID, error) {
	h := md5.New()
	_, err := io.WriteString(h, data)
	if err != nil {
		log.Printf("Error creating hash for %s", data)
		return 0, err
	}
	digest := h.Sum(nil)

	result1 := uint32(digest[3]) << 24
	result2 := uint32(digest[2]) << 16
	result3 := uint32(digest[1]) << 8
	result4 := uint32(digest[0])

	id := result1 | result2 | result3 | result4
	if DebugMode {
		log.Printf("HashRingID for %s is %d", data, id)
	}

	return HashRingID(id), nil
}

// GetBackendsForMetric returns the StatsDBackend instances that are responsible for a metric.
// Responsible in this case means either the ID of the metric is lower than
// the Ring ID of the backend. Or the first backend if the metric ID is higher
// than all backend IDs. This case is the ring wrap around part of the hash
// ring.
// If mirroring is enabled, all backend instances are returned, whatever the metric.
// Accepts a metric name as a string as parameter
// Returns a list of pointers to StatsDBackend instances and error
func (ring *HashRing) GetBackendsForMetric(name string) ([]*StatsDBackend, error) {
	if len(ring.Backends) == 0 {
		return []*StatsDBackend{}, errors.New("no backends in the hashring")
	}

	if ring.Mirror {
		if DebugMode {
			log.Printf("Mirroring is enabled, returning all backends from %v", *ring)
		}
		return ring.Backends, nil
	}

	backends := []*StatsDBackend{ring.Backends[0]}

	metricID, err := GetHashRingPosition(name)
	if err != nil {
		msg := fmt.Sprintf("Unable to get hashring position for %s", name)
		return nil, errors.New(msg)
	}
	if DebugMode {
		log.Printf("Choosing backend from %v", *ring)
	}
	for _, possibleBackend := range ring.Backends {
		if possibleBackend.Alive() && metricID < possibleBackend.RingID {
			// we only set the backend if it has a higher RingID and is alive
			backends[0] = possibleBackend
		}

	}
	if DebugMode {
		log.Printf("Backend for %s is %d", name, backends[0].Port)
	}

	return backends, nil
}

// ByHashRingID implements the sort interface so StatsDBackend instances are sortable by
// hashring ID
type ByHashRingID []*StatsDBackend

func (a ByHashRingID) Len() int           { return len(a) }
func (a ByHashRingID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByHashRingID) Less(i, j int) bool { return a[i].RingID < a[j].RingID }
