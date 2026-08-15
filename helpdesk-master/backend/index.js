import express from "express";
import cors from "cors";
import dotenv from "dotenv";
import morgan from "morgan";
import bodyParser from "body-parser";
import cookieParser from "cookie-parser";
import Auth from "./middleware/auth.js";
import knowledgeBaseRoutes from "./routes/knowledgeBaseRoutes.js";
import userRoutes from "./routes/userRoutes.js";
import ticketRoutes from "./routes/ticketRoutes.js";
import commentsRoutes from "./routes/commentsRoutes.js";
import agentRoutes from "./routes/agents.js";
import videoRoutes from "./routes/videoRoutes.js";
import {swaggerSpec, swaggerUi} from "./swagger.js";
import UserModel from "./models/userModel.js";
import TicketModel from "./models/ticketModel.js";
import CommentModel from "./models/commentsModel.js";

const app = express();
dotenv.config();

// ── Fail-fast startup validation (SG-SEC-2026-08) ──────────────────────────
// In production the helpdesk refuses to start unless the shared identity
// secret and every mandatory database variable are present.
const NODE_ENV = process.env.NODE_ENV || "development";
const registrySecret = process.env.STATGATE_REGISTRY_JWT_SECRET || process.env.SECRETKEY;
if (NODE_ENV === "production") {
  const required = [
    ["SECRETKEY / STATGATE_REGISTRY_JWT_SECRET", registrySecret],
    ["POSTGRES_PASSWORD", process.env.POSTGRES_PASSWORD],
    ["POSTGRES_USER", process.env.POSTGRES_USER],
    ["POSTGRES_DB", process.env.POSTGRES_DB],
  ];
  const missing = required.filter(([, v]) => !v).map(([n]) => n);
  if (missing.length > 0) {
    console.error(`FATAL: required production secrets missing: ${missing.join(", ")}. Startup aborted.`);
    process.exit(1);
  }
}

app.use(express.json({ limit: "10kb" }));
app.use("/uploads", express.static("uploads"));

if (NODE_ENV === "development") app.use(morgan("dev"));

import { connectDB, sequelize } from "./config/db.js";

app.use("/api-docs", swaggerUi.serve, swaggerUi.setup(swaggerSpec));


// ── CORS: restricted to configured origins (never a wildcard) ──────────────
const allowedOrigins = (process.env.CORS_ALLOWED_ORIGINS || "http://localhost:3005,http://localhost:3000")
  .split(",")
  .map((o) => o.trim())
  .filter(Boolean);
app.use(cors({
  origin: (origin, cb) => {
    if (!origin || allowedOrigins.includes(origin)) return cb(null, true);
    return cb(null, false);
  },
  credentials: true,
  methods: ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"],
  allowedHeaders: ["Origin", "Content-Type", "Authorization", "Accept", "X-Request-ID"],
}));
app.use(express.json());
app.use(cookieParser());

app.use(bodyParser.json());
app.use(bodyParser.urlencoded({ extended: false }));

app.use("/api/users", userRoutes);
app.use("/api/t/tickets", ticketRoutes);
app.use("/api/t/comments", commentsRoutes);
app.use("/api/t/agents", agentRoutes);
app.use("/api/videos", videoRoutes);
app.use("/api/knowledge-base", knowledgeBaseRoutes);

// Enterprise search endpoint for unified search (authenticated).
app.get("/api/search", Auth, async (req, res) => {
  try {
    const query = req.query.q;
    if (!query) {
      return res.status(400).json({ status: "error", message: "Search query required" });
    }

    const { Op } = await import("sequelize");
    const searchTerm = `%${query}%`;

    const tickets = await TicketModel.findAll({
      where: {
        [Op.or]: [
          { reportedby: { [Op.iLike]: searchTerm } },
          { facility: { [Op.iLike]: searchTerm } },
          { system: { [Op.iLike]: searchTerm } },
          { category: { [Op.iLike]: searchTerm } },
          { description: { [Op.iLike]: searchTerm } },
          { status: { [Op.iLike]: searchTerm } },
        ],
      },
      limit: 10,
      include: [{ model: UserModel, attributes: { exclude: ["password"] } }],
    });

    const results = tickets.map((ticket) => ({
      type: "ticket",
      id: String(ticket.id),
      title: ticket.description ? ticket.description.substring(0, 80) : `Ticket #${ticket.id}`,
      description: `${ticket.facility} • ${ticket.category} • ${ticket.status}`,
      meta: ticket.system,
    }));

    res.status(200).json(results);
  } catch (error) {
    res.status(500).json({
      status: "error",
      message: "Search failed",
    });
  }
});

app.listen(process.env.PORT, async () => {
    console.log(
        `🚀 Server started successfully on port ${process.env.PORT} in ${NODE_ENV}`
    );

    try {
        await connectDB();

        await UserModel.sync({ force: false, alter: false });
        await TicketModel.sync({ force: false, alter: false });
        await CommentModel.sync({ force: false, alter: false });

        console.log("✅ Synced database successfully...");
    } catch (error) {
        console.error("❌ Database initialization failed:", error.message);
        process.exit(1);
    }
});
