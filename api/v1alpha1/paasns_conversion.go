/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha1

import (
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha2"
	"github.com/rs/zerolog/log"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertFrom converts the Hub version (v1alpha2) to this Paas (v1alpha1).
func (p *PaasNS) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*v1alpha2.PaasNS)
	if !ok {
		return fmt.Errorf("cannot convert to v1alpha1: got %T", srcRaw)
	}

	logger := log.With().
		Any("conversion", p.GetObjectKind().GroupVersionKind()).
		Str("name", p.GetName()).
		Logger()
	logger.Debug().Msg("Starting conversion from hub (v1alpha2) to spoke (v1alpha1)")

	p.ObjectMeta = src.ObjectMeta
	// Deprecated: not required once paas controller is managing the PaasNS resources.
	// The `metadata.name` of the Paas which created the namespace in which this PaasNS is applied
	p.Spec.Paas = src.GetObjectMeta().GetName()
	p.Spec.Groups = src.Spec.Groups
	p.Spec.SSHSecrets = src.Spec.Secrets
	p.Status.Conditions = src.Status.Conditions
	// Deprecated: use paasns.status.conditions instead
	// p.Status.Messages = ...

	return nil
}

// ConvertTo converts this Paas (v1alpha1) to the Hub version (v1alpha2).
func (p *PaasNS) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*v1alpha2.PaasNS)
	if !ok {
		return fmt.Errorf("cannot convert from v1alpha1: got %T", dstRaw)
	}

	logger := log.With().
		Any("conversion", p.GetObjectKind().GroupVersionKind()).
		Str("name", p.GetName()).
		Logger()
	logger.Debug().Msg("Starting conversion from spoke (v1alpha1) to hub (v1alpha2)")

	dst.ObjectMeta = p.ObjectMeta
	dst.Spec.Groups = p.Spec.Groups
	dst.Spec.Secrets = p.Spec.SSHSecrets
	dst.Status.Conditions = p.Status.Conditions
	// How to convert Messages ([]string) to Conditions ([]metav1.Condition)?
	// dst.Status.Conditions = ...

	return nil
}
