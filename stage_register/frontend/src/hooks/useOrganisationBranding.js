import { useEffect, useState } from "react";
import { UsersApi } from "../helpers/api/users";

const DEFAULT_BRANDING = {
  display_name: "StatGate",
  logo_url: "",
  primary_color: "#0f766e",
  secondary_color: "#0f3d3e",
};

export default function useOrganisationBranding() {
  const [branding, setBranding] = useState(DEFAULT_BRANDING);

  useEffect(() => {
    let mounted = true;
    UsersApi.getBranding()
      .then((data) => {
        if (!mounted || !data) return;
        setBranding({
          display_name: data.display_name || DEFAULT_BRANDING.display_name,
          logo_url: data.logo_url || "",
          primary_color: data.primary_color || DEFAULT_BRANDING.primary_color,
          secondary_color: data.secondary_color || DEFAULT_BRANDING.secondary_color,
        });
      })
      .catch(() => {
        if (mounted) setBranding(DEFAULT_BRANDING);
      });
    return () => {
      mounted = false;
    };
  }, []);

  return branding;
}
