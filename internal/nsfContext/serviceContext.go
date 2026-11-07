package nsfContext

import (
	"encoding/json"
	"fmt"
	"nsf/internal/db"
	"nsf/logger"
	"nsf/model"
	"sync"
)

type serviceContextIE struct {
	db.DbIf

	*logger.NsfLogger
}

type serviceContext struct {
	db.DbIf
	mu sync.Mutex

	*logger.NsfLogger
}

func newServiceContext(serviceContextIE *serviceContextIE) *serviceContext {
	return &serviceContext{
		DbIf: serviceContextIE.DbIf,

		NsfLogger: serviceContextIE.NsfLogger,
	}
}

func (s *serviceContext) release() {
	s.NsfLogger.SvcLog.Infoln("Release serviceContext...")

	s.NsfLogger.SvcLog.Infoln("serviceContext released")
}

func (s *serviceContext) registerServices(nfName, sbiAddr string, sbiPort int, services []string) error {
	s.NsfLogger.SvcLog.Tracef("Registering %s services", nfName)
	s.mu.Lock()
	defer s.mu.Unlock()

	data := make(map[string]string, len(services))
	for _, service := range services {
		value, err := json.Marshal(model.NnsfDiscGetResponse{
			Service: service, NfName: nfName, SbiAddress: sbiAddr, SbiPort: sbiPort,
		})
		if err != nil {
			return fmt.Errorf("encode service %q: %w", service, err)
		}
		data[service] = string(value)
	}
	return s.SaveAll(data)
}

func (s *serviceContext) deregisterServices(nfName string, services []string) error {
	s.NsfLogger.SvcLog.Tracef("De-registering %s services", nfName)
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, service := range services {
		owner, _, _, err := s.discoverService(service)
		if err != nil {
			return err
		}
		if owner != nfName {
			return fmt.Errorf("service %q does not belong to NF %q", service, nfName)
		}
	}
	for _, service := range services {
		if err := s.Remove(service); err != nil {
			return fmt.Errorf("remove service %q: %w", service, err)
		}
	}
	return nil
}

func (s *serviceContext) discoverService(service string) (string, string, int, error) {
	s.NsfLogger.SvcLog.Tracef("Discovering service %s", service)

	value, err := s.Load(service)
	if err != nil {
		return "", "", 0, fmt.Errorf("load service %q: %w", service, err)
	}
	if value == "" {
		return "", "", 0, fmt.Errorf("service %q not found", service)
	}
	var record model.NnsfDiscGetResponse
	if err := json.Unmarshal([]byte(value), &record); err != nil {
		return "", "", 0, fmt.Errorf("decode service %q: %w", service, err)
	}
	return record.NfName, record.SbiAddress, record.SbiPort, nil
}
