import express from "express";
import fs from "fs";
import path from "path";
import {v4 as uuidv4} from "uuid";
import dotenv from "dotenv";
import KnowledgeBase from "../models/knowledgeBaseModel.js";
import User from "../models/userModel.js";
import multer from "multer";
import Auth from "../middleware/auth.js";

dotenv.config();

const router = express.Router();
const FILE_STORAGE_PATH = process.env.FILE_STORAGE_PATH || './uploads';

// Ensure file storage path exists
if (!fs.existsSync(FILE_STORAGE_PATH)) {
    fs.mkdirSync(FILE_STORAGE_PATH, {recursive: true});
}

const storage = multer.memoryStorage();

const upload = multer({
    storage,
    fileFilter: (req, file, cb) => {
        if (file.mimetype === "application/pdf") {
            cb(null, true);
        } else {
            cb(new Error("Only PDF files are allowed"));
        }
    },
});



router.post("/", Auth, upload.single("file"), async (req, res) => {
    try {
        const { title, description, createdby } = req.body;

        const missingFields = [];
        if (!req.file) missingFields.push("file");
        if (!title) missingFields.push("title");
        // if (!description) missingFields.push("description");

        if (missingFields.length > 0) {
            return res.status(400).json({
                message: "Missing required fields",
                missingFields,
            });
        }

        const filename = `${uuidv4()}.pdf`;
        const filePath = path.join(FILE_STORAGE_PATH, filename);
        fs.writeFileSync(filePath, req.file.buffer);

        const newEntry = await KnowledgeBase.create({
            file: filename,
            title,
            description,
            createdby: 1,
            createdAt: new Date(),
        });

        res.status(201).json(newEntry);
    } catch (err) {
        console.error("Create error:", err);
        res.status(500).json({ message: "Internal server error" });
    }
});

router.get("/", async (req, res) => {
    try {
        const page = parseInt(req.query.page) || 1;
        const pageSize = parseInt(req.query.pageSize) || 10;
        const offset = (page - 1) * pageSize;

        const {count, rows} = await KnowledgeBase.findAndCountAll({
            offset,
            limit: pageSize,
            include: [{model: User, attributes: ["id", "firstname", "lastname", "email"]}],
            order: [["createdAt", "DESC"]],
        });

        // Add public URL for each PDF file
        const rowsWithUrl = rows.map(row => {
            const rowData = row.get({ plain: true });
            rowData.pdfUrl = `/uploads/${rowData.file}`;
            return rowData;
        });

        res.json({
            total: count,
            page,
            pageSize,
            data: rowsWithUrl,
        });
    } catch (err) {
        console.error("Fetch error:", err);
        res.status(500).json({message: "Internal server error"});
    }
});

router.get("/:id", async (req, res) => {
    try {
        const entry = await KnowledgeBase.findByPk(req.params.id, {
            include: [{model: User, attributes: ["id", "firstname", "lastname", "email"]}],
        });

        if (!entry) {
            return res.status(404).json({message: "Entry not found"});
        }

        // Adding public URL for the PDF file
        const entryData = entry.get({ plain: true });
        entryData.pdfUrl = `/uploads/${entryData.file}`;

        res.json(entryData);
    } catch (err) {
        console.error("Fetch by ID error:", err);
        res.status(500).json({message: "Internal server error"});
    }
});

router.delete("/:id", Auth, async (req, res) => {
    try {
        const entry = await KnowledgeBase.findByPk(req.params.id);
        if (!entry) {
            return res.status(404).json({message: "Entry not found"});
        }

        // Delete file from disk
        const filePath = path.join(FILE_STORAGE_PATH, entry.file);
        if (fs.existsSync(filePath)) {
            fs.unlinkSync(filePath);
        }

        await entry.destroy();
        res.json({message: "Entry deleted successfully"});
    } catch (err) {
        console.error("Delete error:", err);
        res.status(500).json({message: "Internal server error"});
    }
});

export default router;
