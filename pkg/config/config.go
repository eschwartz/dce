package config

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"log"
	"os"
	"reflect"

	"github.com/caarlos0/env"
	"github.com/mitchellh/mapstructure"
)

// ConfigurationError is an error that is returned by configuration
// methods when keys cannot be found or when there is an error whilst
// building the configuration.
type ConfigurationError error

// GenericConfiguration is a generic structure that contains configuration
type configurationValues struct {
	services []interface{}
	types    []reflect.Type
	impls    []reflect.Value
	vals     map[string]interface{}
	envKeys  map[string]string
}

// ConfigurationBuilder is the default implementation of a configuration loader.
type ConfigurationBuilder struct {
	values  *configurationValues
	parsers env.CustomParsers
	isBuilt bool
}

// Unmarshal loads configuration into the provided structure from environment variables.
// Use the "env" tag on cfgStruct fields to indicate the corresponding environment variable to load from.
func (config *ConfigurationBuilder) Unmarshal(cfgStruct interface{}) error {
	// Unmarshal env vars into the struct
	config.parsers = config.createCustomParsers()
	err := env.ParseWithFuncs(cfgStruct, config.parsers)
	if err != nil {
		return err
	}

	// Unmarshal services into the struct
	val := reflect.ValueOf(cfgStruct)
	// Reflection will panic on nil cfgStruct, so better to catch it here
	if val.IsNil() {
		return errors.New("Unable to unmarshal config: the provided struct is nil")
	}
	if config.values != nil {
		elem := val.Elem()
		// Look at all the fields in our target struct
		for fi := 0; fi < elem.NumField(); fi++ {
			field := elem.Field(fi)
			fieldType := field.Type()

			// See if we have a matching service type
			for ti, t := range config.values.types {
				// Target struct has a pointer to our service value
				if fieldType.Kind() == reflect.Interface && t.Implements(fieldType) {
					field.Set(config.values.impls[ti])
					break
				} else if fieldType.Kind() == reflect.Ptr && fieldType.Elem() == t {
					// Convert our value to a pointer
					impl := config.values.impls[ti]
					ptrVal := reflect.New(impl.Type())
					ptrVal.Elem().Set(impl)
					field.Set(ptrVal)
					break
				} else if t.Kind() == reflect.Ptr && t.Elem() == fieldType {
					// Target struct has a value, for our service pointer
					field.Set(config.values.impls[ti].Elem())
					break
				} else if t == fieldType {
					// If the type matches, set the value
					field.Set(config.values.impls[ti])
					break
				}
			}
		}
	}

	return nil
}

// Dump dumps the current config into the provided structure. Config keys are matched to
// cfgStruct fields using the "env" tag.
func (config *ConfigurationBuilder) Dump(cfgStruct interface{}) error {
	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			TagName: "env",
			Result:  cfgStruct,
		})
	if err != nil {
		panic(err)
	}
	err = decoder.Decode(config.values.vals)
	return err
}

// WithService is a Builder Pattern method that allows you to specify services
// for the given type.
func (config *ConfigurationBuilder) WithService(svc interface{}) *ConfigurationBuilder {
	config.initialize()
	config.values.services = append(config.values.services, svc)
	config.values.types = append(config.values.types, reflect.TypeOf(svc))
	config.values.impls = append(config.values.impls, reflect.ValueOf(svc))
	return config
}

// WithEnv allows you to point to an environment variable for the value and
// also specify a default using defaultValue
func (config *ConfigurationBuilder) WithEnv(key string, envVar string, defaultValue interface{}) *ConfigurationBuilder {
	config.initialize()

	envVal, ok := os.LookupEnv(envVar)

	if !ok {
		config.values.vals[key] = defaultValue
	} else {
		config.values.vals[key] = envVal
	}

	return config
}

// ParameterStoreVal stores the parameters for a SSM Value
type ParameterStoreVal struct {
	Key           string
	ParameterName string
	DefaultValue  string
}

// WithParameterStoreEnv sets a config value from SSM Parameter store. The Parameter name is taken
// from the provided environment variable. If the environment variable or SSM parameter can't be retrieved,
// then the default value is used.
// Requires that an SSM service of type ssmiface.SSMAPI is contained within config
func (config *ConfigurationBuilder) WithParameterStoreEnv(key string, envVar string, defaultValue string) *ConfigurationBuilder {
	config.initialize()

	envVal, ok := os.LookupEnv(envVar)

	if !ok {
		config.values.vals[key] = defaultValue
	} else {
		config.values.vals[key] = ParameterStoreVal{
			Key:           key,
			ParameterName: envVal,
			DefaultValue:  defaultValue,
		}
	}

	return config
}

// WithVal allows you to hardcode string values into the configuration.
// This is good for testing, injecting known values or values derived by means
// outside the configuration.
func (config *ConfigurationBuilder) WithVal(key string, val interface{}) *ConfigurationBuilder {
	config.initialize()
	config.values.vals[key] = val
	return config
}

// GetService retreives the service with the given type. An error is thrown if
// the service is not found.
func (config *ConfigurationBuilder) GetService(svcFor interface{}) error {
	k := reflect.TypeOf(svcFor).Elem()
	kind := k.Kind()
	if kind == reflect.Ptr {
		k = k.Elem()
		kind = k.Kind()
	}
	for i, t := range config.values.types {
		if kind == reflect.Interface && t.Implements(k) {
			reflect.Indirect(
				reflect.ValueOf(svcFor),
			).Set(config.values.impls[i])
			return nil
		} else if kind == reflect.Struct && k.AssignableTo(t.Elem()) {
			reflect.ValueOf(svcFor).Elem().Set(config.values.impls[i].Elem())
			return nil
		}
	}
	return ConfigurationError(fmt.Errorf("no service found in configuration for key type: %s", k))
}

// GetStringVal returns the value of the key as a string.
func (config *ConfigurationBuilder) GetStringVal(key string) (string, error) {
	if !config.isBuilt {
		return "", ConfigurationError(errors.New("call Build() before attempting to get values"))
	}

	val, ok := config.values.vals[key]

	if !ok {
		return "", ConfigurationError(fmt.Errorf("no value found in configuration for key: %s", key))
	}

	return val.(string), nil
}

// GetVal returns the raw value
func (config *ConfigurationBuilder) GetVal(key string) (interface{}, error) {
	if !config.isBuilt {
		return "", ConfigurationError(errors.New("call Build() before attempting to get values"))
	}

	val, ok := config.values.vals[key]

	if !ok {
		return nil, ConfigurationError(fmt.Errorf("no value found in configuration for key: %s", key))
	}

	return val, nil
}

// Build builds the configuration.
func (config *ConfigurationBuilder) Build() error {
	// Add any "expensive" operations here. Validations, type conversions, etc.
	// We already have basic maps.

	config.isBuilt = true
	return nil
}

func (config *ConfigurationBuilder) initialize() {
	if config.values == nil {
		config.values = &configurationValues{}
	}
	if config.values.envKeys == nil {
		config.values.envKeys = make(map[string]string)
	}
	if config.values.vals == nil {
		config.values.vals = make(map[string]interface{})
	}
}

func (config *ConfigurationBuilder) createCustomParsers() env.CustomParsers {
	funcMap := env.CustomParsers{}
	return funcMap
}

// RetrieveParameterStoreVals - Get the values from the AWS Parameter Store
func (config *ConfigurationBuilder) RetrieveParameterStoreVals() error {

	// Detect values that need to be retrieved from SSM
	valsToRetrieve := map[string]ParameterStoreVal{}
	for _, val := range config.values.vals {
		if _, ok := val.(ParameterStoreVal); ok {
			paramName := string(val.(ParameterStoreVal).ParameterName)
			valsToRetrieve[paramName] = val.(ParameterStoreVal)
		}
	}

	if len(valsToRetrieve) != 0 {
		// config must contain an SSM service to retrieve vals from SSM
		var ssmClient ssmiface.SSMAPI
		if err := config.GetService(&ssmClient); err != nil {
			return err
		}

		// Using bulk api to reduce number of SSM requests
		withDecryption := false
		getParametersOutput, err := ssmClient.GetParameters(&ssm.GetParametersInput{
			Names:          getKeyPtrs(valsToRetrieve),
			WithDecryption: &withDecryption,
		})
		if err != nil {
			return err
		}

		// Overwrite config.values.vals {Config Key: Param Name} -> {Config Key: Param Value}
		params := getParametersOutput.Parameters
		for _, param := range params {
			log.Print("Retrieved SSM Parameter: ", param.GoString())
			key := valsToRetrieve[*param.Name].Key
			config.WithVal(key, *param.Value)
		}

		invalidParams := getParametersOutput.InvalidParameters
		for _, invalidParam := range invalidParams {
			log.Print("Invalid SSM Parameter: ", invalidParam)
			key := valsToRetrieve[*invalidParam].Key
			defaultVal := valsToRetrieve[*invalidParam].DefaultValue
			config.WithVal(key, defaultVal)
		}
	}
	return nil
}

func getKeyPtrs(aMap map[string]ParameterStoreVal) []*string {
	keys := []*string{}
	for k := range aMap {
		newK := k
		keys = append(keys, &newK)
	}
	return keys
}
