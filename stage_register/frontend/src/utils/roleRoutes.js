export function getRoleRoute(role) {
  switch (role) {
    case "admin":
      return "/dashboard";
    case "public":
      return "/initiator/requests";
    case "district":
      return "/district/dashboard";
    case "district_initiator":
      return "/initiator/requests";
    case "district_approver":
      return "/reviewer/dashboard";
    case "moh_clinical":
      return "/reviewer/dashboard";
    case "moh_publisher":
      return "/reviewer/dashboard";
    default:
      return "/";
  }
}

