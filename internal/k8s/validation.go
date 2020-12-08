package k8s

import (
	"errors"
	"fmt"
	"sort"

	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	networking "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	mergeableIngressTypeAnnotation       = "nginx.org/mergeable-ingress-type"
	lbMethodAnnotation                   = "nginx.org/lb-method"
	healthChecksAnnotation               = "nginx.com/health-checks"
	healthChecksMandatoryAnnotation      = "nginx.com/health-checks-mandatory"
	healthChecksMandatoryQueueAnnotation = "nginx.com/health-checks-mandatory-queue"
)

type annotationValidationContext struct {
	annotations map[string]string
	name        string
	value       string
	isPlus      bool
	fieldPath   *field.Path
}

type annotationValidationFunc func(context *annotationValidationContext) field.ErrorList
type validatorFunc func(val string) error

var (
	// annotationValidations defines the various validations which will be applied in order to each ingress annotation.
	// If any specified validation fails, the remaining validations for that annotation will not be run.
	annotationValidations = map[string][]annotationValidationFunc{
		mergeableIngressTypeAnnotation: {
			validateRequiredAnnotation,
			validateMergeableIngressTypeAnnotation,
		},
		lbMethodAnnotation: {
			validateRequiredAnnotation,
			validateLBMethodAnnotation,
		},
		healthChecksAnnotation: {
			validateRequiredAnnotation,
			validatePlusOnlyAnnotation,
			validateBoolAnnotation,
		},
		healthChecksMandatoryAnnotation: {
			validateRelatedAnnotation(healthChecksAnnotation, validateIsTrue),
			validateRequiredAnnotation,
			validateBoolAnnotation,
		},
		healthChecksMandatoryQueueAnnotation: {
			validateRelatedAnnotation(healthChecksMandatoryAnnotation, validateIsTrue),
			validateRequiredAnnotation,
			validateNonNegativeIntAnnotation,
		},
	}
	annotationNames = sortedAnnotationNames(annotationValidations)
)

func sortedAnnotationNames(annotationValidations map[string][]annotationValidationFunc) []string {
	sortedNames := make([]string, 0)
	for annotationName := range annotationValidations {
		sortedNames = append(sortedNames, annotationName)
	}
	sort.Strings(sortedNames)
	return sortedNames
}

// validateIngress validate an Ingress resource with rules that our Ingress Controller enforces.
// Note that the full validation of Ingress resources is done by Kubernetes.
func validateIngress(ing *networking.Ingress, isPlus bool) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateIngressAnnotations(ing.Annotations, isPlus, field.NewPath("annotations"))...)

	allErrs = append(allErrs, validateIngressSpec(&ing.Spec, field.NewPath("spec"))...)

	if isMaster(ing) {
		allErrs = append(allErrs, validateMasterSpec(&ing.Spec, field.NewPath("spec"))...)
	} else if isMinion(ing) {
		allErrs = append(allErrs, validateMinionSpec(&ing.Spec, field.NewPath("spec"))...)
	}

	return allErrs
}

func validateIngressAnnotations(annotations map[string]string, isPlus bool, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for _, name := range annotationNames {
		if value, nameExists := annotations[name]; nameExists {
			if validationFuncs, validationExists := annotationValidations[name]; validationExists {
				for _, validationFunc := range validationFuncs {
					valErrors := validationFunc(&annotationValidationContext{
						annotations: annotations,
						name:        name,
						value:       value,
						isPlus:      isPlus,
						fieldPath:   fieldPath.Child(name),
					})
					if len(valErrors) > 0 {
						allErrs = append(allErrs, valErrors...)
						break
					}
				}
			}
		}
	}

	return allErrs
}

func validateRelatedAnnotation(name string, validator validatorFunc) annotationValidationFunc {
	return func(context *annotationValidationContext) field.ErrorList {
		allErrs := field.ErrorList{}
		val, exists := context.annotations[name]
		if !exists {
			return append(allErrs, field.Forbidden(context.fieldPath, fmt.Sprintf("related annotation %s: must be set", name)))
		}

		if err := validator(val); err != nil {
			return append(allErrs, field.Forbidden(context.fieldPath, fmt.Sprintf("related annotation %s: %s", name, err.Error())))
		}
		return allErrs
	}
}

func validateMergeableIngressTypeAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if context.value != "master" && context.value != "minion" {
		return append(allErrs, field.Invalid(context.fieldPath, context.value, "must be one of: 'master' or 'minion'"))
	}
	return allErrs
}

func validateLBMethodAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if context.isPlus {
		if _, err := configs.ParseLBMethodForPlus(context.value); err != nil {
			return append(allErrs, field.Invalid(context.fieldPath, context.value, err.Error()))
		}
	} else {
		if _, err := configs.ParseLBMethod(context.value); err != nil {
			return append(allErrs, field.Invalid(context.fieldPath, context.value, err.Error()))
		}
	}
	return allErrs
}

func validateRequiredAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if context.value == "" {
		return append(allErrs, field.Required(context.fieldPath, ""))
	}
	return allErrs
}

func validatePlusOnlyAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if !context.isPlus {
		return append(allErrs, field.Forbidden(context.fieldPath, "annotation requires NGINX Plus"))
	}
	return allErrs
}

func validateBoolAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, err := configs.ParseBool(context.value); err != nil {
		return append(allErrs, field.Invalid(context.fieldPath, context.value, "must be a valid boolean"))
	}
	return allErrs
}

func validateNonNegativeIntAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, err := configs.ParseUint64(context.value); err != nil {
		return append(allErrs, field.Invalid(context.fieldPath, context.value, "must be a non-negative integer"))
	}
	return allErrs
}

func validateIsTrue(v string) error {
	b, err := configs.ParseBool(v)
	if err != nil {
		return err
	}
	if !b {
		return errors.New("must be true")
	}
	return nil
}

func validateIngressSpec(spec *networking.IngressSpec, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allHosts := sets.String{}

	if len(spec.Rules) == 0 {
		return append(allErrs, field.Required(fieldPath.Child("rules"), ""))
	}

	for i, r := range spec.Rules {
		idxPath := fieldPath.Child("rules").Index(i)

		if r.Host == "" {
			allErrs = append(allErrs, field.Required(idxPath.Child("host"), ""))
		} else if allHosts.Has(r.Host) {
			allErrs = append(allErrs, field.Duplicate(idxPath.Child("host"), r.Host))
		} else {
			allHosts.Insert(r.Host)
		}
	}

	return allErrs
}

func validateMasterSpec(spec *networking.IngressSpec, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(spec.Rules) != 1 {
		return append(allErrs, field.TooMany(fieldPath.Child("rules"), len(spec.Rules), 1))
	}

	// the number of paths of the first rule of the spec must be 0
	if spec.Rules[0].HTTP != nil && len(spec.Rules[0].HTTP.Paths) > 0 {
		pathsField := fieldPath.Child("rules").Index(0).Child("http").Child("paths")
		return append(allErrs, field.TooMany(pathsField, len(spec.Rules[0].HTTP.Paths), 0))
	}

	return allErrs
}

func validateMinionSpec(spec *networking.IngressSpec, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(spec.TLS) > 0 {
		allErrs = append(allErrs, field.TooMany(fieldPath.Child("tls"), len(spec.TLS), 0))
	}

	if len(spec.Rules) != 1 {
		return append(allErrs, field.TooMany(fieldPath.Child("rules"), len(spec.Rules), 1))
	}

	// the number of paths of the first rule of the spec must be greater than 0
	if spec.Rules[0].HTTP == nil || len(spec.Rules[0].HTTP.Paths) == 0 {
		pathsField := fieldPath.Child("rules").Index(0).Child("http").Child("paths")
		return append(allErrs, field.Required(pathsField, "must include at least one path"))
	}

	return allErrs
}
