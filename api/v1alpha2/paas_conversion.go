/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha2

import (
	"fmt"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this Paas (v1alpha2) to the Hub version (v1alpha1).
func (p *Paas) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*v1alpha1.Paas)
	if !ok {
		return fmt.Errorf("cannot convert to %s/%s: must be v1alpha1", dst.Namespace, dst.Name)
	}

	// TODO(AxiomaticFixedChimpanzee): Implement conversion logic from v1alpha2 to v1alpha1
	return nil
}

// ConvertFrom converts the Hub version (v1alpha1) to this Paas (v1alpha2).
func (p *Paas) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*v1alpha1.Paas)
	if !ok {
		return fmt.Errorf("cannot convert %s/%s: must be v1alpha1", src.Namespace, src.Name)
	}

	// TODO(AxiomaticFixedChimpanzee): Implement conversion logic from v1alpha1 to v1alpha2
	return nil
}
