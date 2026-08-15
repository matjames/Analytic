import express from "express";
import bcrypt from "bcrypt";
import jwt from "jsonwebtoken";
import multer from "multer";
import Auth, { requireAdmin, toSafeUser } from "../middleware/auth.js";
import UserModel from "../models/userModel.js";

const router = express.Router();
const upload = multer({
    dest: "uploads/",
    limits: { fileSize: 1024 * 1024 * 5 }, // 5 MB
});

// ── Role whitelist for PUBLIC registration (SG-SEC-2026-08). Privileged
// roles (admin, superadmin, platform_admin, operator) are provisioned by an
// administrator; a self-registered user may never hold them.
const PUBLIC_ROLES = new Set(["user", "viewer", "agent"]);

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;
const passwordPolicy = (pw) =>
    typeof pw === "string" &&
    pw.length >= 10 &&
    /[A-Z]/.test(pw) &&
    /[a-z]/.test(pw) &&
    /[0-9]/.test(pw);

// helper: create jwt for a user
const signUserToken = (user) =>
    jwt.sign({ id: user.id }, process.env.SECRETKEY || process.env.STATGATE_REGISTRY_JWT_SECRET, {
        expiresIn: 86400, // 24h
        algorithm: "HS256",
    });

// catch-all safe error responder (no raw stack traces / DB text)
const respondSafeError = (res, status, message) =>
    res.status(status).json({ status: "error", message });

router.post("/register", upload.single("profilePicture"), async (req, res) => {
  try {
    const {
      username,
      email,
      role,
      password,
      firstname,
      lastname,
      phoneNo,
      system,
    } = req.body;

    // Role whitelist — never accept a privileged role at registration.
    const requestedRole = (role || "user").toLowerCase().trim();
    if (!PUBLIC_ROLES.has(requestedRole)) {
      return res.status(403).json({ status: "error", message: "Role not permitted for registration" });
    }

    // Email + password policy validation.
    if (!EMAIL_RE.test(email || "")) {
      return res.status(400).json({ status: "error", message: "A valid email address is required" });
    }
    if (!passwordPolicy(password)) {
      return res.status(400).json({ status: "error", message: "Password must be at least 10 characters with upper and lowercase letters and a digit" });
    }

    // Duplicate prevention.
    const existing = await UserModel.findOne({ where: { email: (email || "").toLowerCase() } });
    if (existing) {
      return res.status(409).json({ status: "error", message: "An account with this email already exists" });
    }

    const profilePicture = req.file ? `/uploads/${req.file.filename}` : null;

    const data = {
      username,
      email: (email || "").toLowerCase(),
      role: requestedRole,
      firstname,
      lastname,
      phoneNo,
      system,
      profilePicture,
      password: await bcrypt.hash(password, 12),
    };

    const user = await UserModel.create(data);

    if (user) {
      const token = signUserToken(user);

      res.cookie("jwt", token, { maxAge: 24 * 60 * 60 * 1000, httpOnly: true, sameSite: "lax" });
      // Safe serialization: never return or log the password hash.
      return res.status(201).json({ status: "success", user: toSafeUser(user.get({ plain: true })), token });
    }
    return res.status(409).json({ status: "error", message: "Details are not correct" });
  } catch (error) {
    if (error && error.name === "SequelizeUniqueConstraintError") {
      return res.status(409).json({ status: "error", message: "An account with this email already exists" });
    }
    return respondSafeError(res, 500, "Registration failed");
  }
});

router.post("/login", async (req, res) => {
  try {
    const { username, password } = req.body;

    if (!username || !password) {
      return res.status(400).json({ status: "error", message: "Username and password are required" });
    }

    const user = await UserModel.findOne({
      where: { username },
    });

    if (user) {
      const isSame = await bcrypt.compare(password, user.password);

      if (isSame) {
        const token = signUserToken(user);
        return res.status(200).json({ status: "success", accessToken: token, user: toSafeUser(user.get({ plain: true })) });
      }
      return res.status(401).json({ status: "error", message: "Authentication failed" });
    }
    return res.status(401).json({ status: "error", message: "Authentication failed" });
  } catch (error) {
    return respondSafeError(res, 500, "Login failed");
  }
});

router.get("/me", Auth, async (req, res) => {
  try {
    const user = await UserModel.findByPk(req.user.id, { attributes: { exclude: ["password"] } });
    res.status(200).json({ success: true, user });
  } catch (error) {
    return respondSafeError(res, 500, "Unable to load user");
  }
});

// List all users — authenticated ADMIN only.
router.get("/", Auth, requireAdmin, async (req, res) => {
  try {
    const users = await UserModel.findAll({ attributes: { exclude: ["password"] } });
    res.status(200).json({
      status: "success",
      results: users.length,
      users,
    });
  } catch (error) {
    return respondSafeError(res, 500, "Unable to list users");
  }
});

// Agent directory — authenticated only.
router.get("/agents", Auth, async (req, res) => {
  try {
    const page = parseInt(req.query.page) || 1;
    const limit = parseInt(req.query.limit) || 10;
    const skip = (page - 1) * limit;

    const totalRecords = await UserModel.count();
    const totalPages = Math.ceil(totalRecords / limit);

    const users = await UserModel.findAll({
      limit,
      offset: skip,
      where: { role: "agent" },
      attributes: { exclude: ["password"] },
    });

    res.status(200).json({
      status: "success",
      results: users.length,
      users,
      totalRecords,
      totalPages,
      currentPage: page,
    });
  } catch (error) {
    return respondSafeError(res, 500, "Unable to load agents");
  }
});

router.patch("/:id", Auth, upload.single("profilePicture"), async (req, res) => {
  // A user may only update their own profile unless they are an admin.
  const isAdmin = ["admin", "superadmin", "platform_admin", "operator"].includes((req.user.role || "").toLowerCase());
  if (String(req.user.id) !== String(req.params.id) && !isAdmin) {
    return res.status(403).json({ status: "error", message: "Admin access required" });
  }

  const { password, role, ...fields } = req.body;

  // Role changes require admin rights.
  if (role && !isAdmin) {
    return res.status(403).json({ status: "error", message: "Role changes require administrator access" });
  }

  try {
    if (password) {
      if (!passwordPolicy(password)) {
        return res.status(400).json({ status: "error", message: "Password must be at least 10 characters with upper/lowercase and a digit" });
      }
      fields.password = await bcrypt.hash(password, 12);
    }

    if (req.file) {
      fields.profilePicture = `/uploads/${req.file.filename}`;
    }

    const result = await UserModel.update(
      { ...fields, updatedAt: Date.now() },
      {
        where: { id: req.params.id },
      }
    );

    if (result[0] === 0) {
      return res.status(404).json({ status: "error", message: "User not found" });
    }

    const user = await UserModel.findByPk(req.params.id, { attributes: { exclude: ["password"] } });

    return res.status(200).json({ status: "success", data: { user } });
  } catch (error) {
    return respondSafeError(res, 500, "Update failed");
  }
});

router.get("/:id", Auth, requireAdmin, async (req, res) => {
  try {
    const note = await UserModel.findByPk(req.params.id, { attributes: { exclude: ["password"] } });

    if (!note) {
      return res.status(404).json({ status: "error", message: "User not found" });
    }

    return res.status(200).json({ status: "success", data: { user: note } });
  } catch (error) {
    return respondSafeError(res, 500, "Unable to load user");
  }
});

router.delete("/:id", Auth, requireAdmin, async (req, res) => {
  try {
    const result = await UserModel.destroy({
      where: { id: req.params.id },
      force: true,
    });

    if (result === 0) {
      return res.status(404).json({ status: "error", message: "User not found" });
    }

    return res.status(204).end();
  } catch (error) {
    return respondSafeError(res, 500, "Delete failed");
  }
});

export default router;
