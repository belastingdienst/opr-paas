/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package v1alpha2

import (
	"log"

	v1alpha1 "github.com/belastingdienst/opr-paas/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this Paas (v1alpha2) to the Hub version (v1alpha1).
func (src *Paas) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha1.Paas)
	log.Printf("ConvertTo: Converting Paas from Spoke version v1alpha2 to Hub version v1alpha1;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// TODO(AxiomaticFixedChimpanzee): Implement conversion logic from v1alpha2 to v1alpha1
	return nil
}

// ConvertFrom converts the Hub version (v1alpha1) to this Paas (v1alpha2).
func (dst *Paas) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.Paas)
	log.Printf("ConvertFrom: Converting Paas from Hub version v1alpha1 to Spoke version v1alpha2;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// TODO(AxiomaticFixedChimpanzee): Implement conversion logic from v1alpha1 to v1alpha2
	return nil
}
