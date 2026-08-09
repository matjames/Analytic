import express from "express";
import cors from "cors";
import dotenv from "dotenv";
import morgan from "morgan";
import bodyParser from "body-parser";
import cookieParser from "cookie-parser";
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

app.use(express.json({ limit: "10kb" }));
app.use("/uploads", express.static("uploads"));

if (process.env.NODE_ENV === "development") app.use(morgan("dev"));

import { connectDB, sequelize } from "./config/db.js";

app.use("/api-docs", swaggerUi.serve, swaggerUi.setup(swaggerSpec));


app.use(cors());
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

// Enterprise search endpoint for unified search
app.get("/api/search", async (req, res) => {
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
      message: error.message,
    });
  }
});

app.listen(process.env.PORT, async () => {
    console.log(
        `🚀Server started Successfully on port ${process.env.PORT} in ${process.env.NODE_ENV}`
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
