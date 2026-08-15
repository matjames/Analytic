import express from "express";
import CommentModel from "../models/commentsModel.js";
import Auth, { requireAdmin } from "../middleware/auth.js";

const router = express.Router();

// Mutations are protected (SG-SEC-2026-08): unauthenticated writes are DENIED.
router.post('/', Auth, async (req, res) => {
    try {
        const comment = await CommentModel.create({
            ...req.body,
            createdBy: req.user ? req.user.id : null,
        });
        res.status(201).json({
            status: "success",
            comment
        });
    } catch (error) {
        res.status(500).json({ status: "error", message: "Failed to create comment" });
    }
});

router.get("/ticket/:id", Auth, async (req, res) => {
    try {
        const comments = await CommentModel.findAll({
            where: { ticketId: req.params.id },
        });

        res.status(200).json({
            status: "success",
            results: comments.length,
            comments,
        });
    } catch (error) {
        res.status(500).json({ status: "error", message: "Failed to load comments" });
    }
});

router.get("/:id", Auth, async (req, res) => {
    try {
        const comment = await CommentModel.findByPk(req.params.id);

        if (!comment) {
            return res.status(404).json({ status: "error", message: "Comment not found" });
        }

        res.status(200).json({ status: "success", comment });
    } catch (error) {
        res.status(500).json({ status: "error", message: "Failed to load comment" });
    }
});

router.delete("/:id", Auth, requireAdmin, async (req, res) => {
    try {
        const result = await CommentModel.destroy({
            where: { id: req.params.id },
            force: true,
        });

        if (result === 0) {
            return res.status(404).json({ status: "error", message: "Comment not found" });
        }

        res.status(204).end();
    } catch (error) {
        res.status(500).json({ status: "error", message: "Failed to delete comment" });
    }
});

export default router;