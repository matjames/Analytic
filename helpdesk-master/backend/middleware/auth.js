import jwt from "jsonwebtoken";
import User from "../models/userModel.js";

// Auth middleware (SG-SEC-2026-08): validates the shared StatGate JWT with
// strict HS256 enforcement. Every mutation endpoint relies on it.
const Auth = async (req, res, next) => {
    let token;

    const secret = process.env.SECRETKEY || process.env.STATGATE_REGISTRY_JWT_SECRET;
    // Never accept the removed/legacy secrets as valid signing keys.
    const legacySecret = ["statgate", "helpdesk", "secret", "2026"].join("_");
    const legacyChat = "statchat-prod-secret";
    if (!secret || secret === legacySecret || secret === legacyChat) {
        return res.status(500).json({ message: "Authentication service is not configured" });
    }

    if (req.headers.authorization && req.headers.authorization.startsWith("Bearer")) {
        token = req.headers.authorization.split(" ")[1];
    } else if (req.cookies && req.cookies.jwt) {
        token = req.cookies.jwt;
    }

    if (!token) {
        return res.status(401).json({ message: "No Access Token Found" });
    }

    try {
        const decoded = jwt.verify(token, secret, { algorithms: ["HS256"] });
        const user = await User.findByPk(decoded.id);

        if (!user) {
            return res.status(404).json({ message: "No User Found with this Id" });
        }
        req.user = user;
        next();
    } catch (error) {
        return res.status(401).json({ message: "Not authorized to access this route" });
    }
};

// requireAdmin restricts a route to users whose role is privileged.
const requireAdmin = (req, res, next) => {
    const role = req.user && (req.user.role || "").toLowerCase();
    if (["admin", "superadmin", "platform_admin", "operator"].includes(role)) {
        return next();
    }
    return res.status(403).json({ message: "Admin access required" });
};

// toSafeUser strips password material from a user object (never log/emit).
export const toSafeUser = (userData) => {
    if (!userData) return userData;
    const safe = { ...userData };
    delete safe.password;
    delete safe.passwordHash;
    return safe;
};

export default Auth;
export { requireAdmin };