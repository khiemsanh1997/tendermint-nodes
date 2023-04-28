package services

import "reflect"

type Service interface {
	OnStart() error
	OnStop() error
}

type CompositeService struct {
	services []Service
}

func NewCompositeService(services ...Service) *CompositeService {
	return &CompositeService{
		services: services,
	}
}

func (cs *CompositeService) OnStart() error {
	for _, service := range cs.services {
		err := service.OnStart()
		if err != nil {
			return err
		}
	}
	return nil
}

func (cs *CompositeService) OnStop() error {
	for _, service := range cs.services {
		err := service.OnStop()
		if err != nil {
			return err
		}
	}
	return nil
}

func (cs *CompositeService) GetServiceByType(serviceType reflect.Type) Service {
	for _, service := range cs.services {
		if reflect.TypeOf(service) == serviceType {
			return service
		}
	}
	return nil
}
