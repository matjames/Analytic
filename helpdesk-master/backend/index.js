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


const app = express();
dotenv.config();

app.use(express.json({ limit: "10kb" }));
app.use("/uploads", express.static("uploads"));

if (process.env.NODE_ENV === "development") app.use(morgan("dev"));

import { connectDB, sequelize } from "./config/db.js";

import "./models/userModel.js";
import "./models/ticketModel.js";
import "./models/commentsModel.js";

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

app.listen(process.env.PORT, async () => {
    console.log(
        `🚀Server started Successfully on port ${process.env.PORT} in ${process.env.NODE_ENV}`
    );
    await connectDB();
    sequelize.sync({ force: false }).then(() => {
        console.log("✅Synced database successfully...");
    });
});
