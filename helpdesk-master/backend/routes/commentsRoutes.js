import express from "express";
import CommentModel from "../models/commentsModel.js";

const router = express.Router();

router.post('/', async (req, res) => {
    try {
        const comment = await CommentModel.create(req.body);
        res.status(201).json({
            status: "success",
            comment
        });
    } catch (error) {
        res.status(500).json({
            status: "error",
            message: error.message,
        });
    }
});

router.get("/ticket/:id", async (req, res) => {
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
        res.status(500).json({
            status: "error",
            message: error.message,
        });
    }
});

router.get("/:id", async (req, res) => {
    try {
        const comment = await CommentModel.findByPk(req.params.id);

        if (!comment) {
            return res.status(404).json({
                status: "fail",
                message: "Ticket with that ID not found",
            });
        }

        res.status(200).json({
            status: "success",
            comment,
        });
    } catch (error) {
        res.status(500).json({
            status: "error",
            message: error.message,
        });
    }
});

router.delete("/:id", async (req, res) => {
    try {
        const result = await CommentModel.destroy({
            where: { id: req.params.id },
            force: true,
        });

        if (result === 0) {
            return res.status(404).json({
                status: "fail",
                message: "Ticket with that ID not found",
            });
        }

        res.status(204).json();
    } catch (error) {
        res.status(500).json({
            status: "error",
            message: error.message,
        });
    }
});

export default router;