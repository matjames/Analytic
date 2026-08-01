import express from "express";
import Auth from "../middleware/auth.js";
import User from "../models/userModel.js";
import TicketModel from "../models/ticketModel.js";

const router = express.Router();

router.get("/", Auth, async (req, res) => {

    const user = await User.findByPk(req.user.id, { attributes: { exclude: ["password"] } });

    try {
        const page = parseInt(req.query.page) || 1;
        const limit = parseInt(req.query.limit) || 10;
        const skip = (page - 1) * limit;

        const filters = {
            ...(req.query.category && { category: req.query.category }),
            ...(req.query.system && { system: req.query.system }),
            ...(req.query.level && { level: req.query.level }),
            ...(req.query.facility && { facility: req.query.facility }),
            agentId: user.id,
            status: 'open'
        };

        const hasFilters = Object.keys(filters).length > 0;
        const whereCondition = hasFilters ? filters : {};

        const totalRecords = await TicketModel.count({ where: whereCondition });
        const totalPages = Math.ceil(totalRecords / limit);

        const tickets = await TicketModel.findAll({
            where: whereCondition,
            limit,
            offset: skip,
            include: [{ model: User }],
        });

        res.status(200).json({
            status: "success",
            results: tickets.length,
            tickets,
            totalRecords,
            totalPages,
            currentPage: page,
        });
    } catch (error) {
        res.status(500).json({
            status: "error",
            message: error.message,
        });
    }
});

router.get("/pending", Auth, async (req, res) => {

    const user = await User.findByPk(req.user.id, { attributes: { exclude: ["password"] } });

    try {
        const page = parseInt(req.query.page) || 1;
        const limit = parseInt(req.query.limit) || 10;
        const skip = (page - 1) * limit;

        const filters = {
            ...(req.query.category && { category: req.query.category }),
            ...(req.query.system && { system: req.query.system }),
            ...(req.query.level && { level: req.query.level }),
            ...(req.query.facility && { facility: req.query.facility }),
            agentId: user.id,
            status: 'pending'
        };

        const hasFilters = Object.keys(filters).length > 0;
        const whereCondition = hasFilters ? filters : {};

        const totalRecords = await TicketModel.count({ where: whereCondition });
        const totalPages = Math.ceil(totalRecords / limit);

        const tickets = await TicketModel.findAll({
            where: whereCondition,
            limit,
            offset: skip,
            include: [{ model: User }],
        });

        res.status(200).json({
            status: "success",
            results: tickets.length,
            tickets,
            totalRecords,
            totalPages,
            currentPage: page,
        });
    } catch (error) {
        res.status(500).json({
            status: "error",
            message: error.message,
        });
    }
});

router.get("/closed", Auth, async (req, res) => {

    const user = await User.findByPk(req.user.id, { attributes: { exclude: ["password"] } });

    try {
        const page = parseInt(req.query.page) || 1;
        const limit = parseInt(req.query.limit) || 10;
        const skip = (page - 1) * limit;

        const filters = {
            ...(req.query.category && { category: req.query.category }),
            ...(req.query.system && { system: req.query.system }),
            ...(req.query.level && { level: req.query.level }),
            ...(req.query.facility && { facility: req.query.facility }),
            agentId: user.id,
            status: 'closed'
        };

        const hasFilters = Object.keys(filters).length > 0;
        const whereCondition = hasFilters ? filters : {};

        const totalRecords = await TicketModel.count({ where: whereCondition });
        const totalPages = Math.ceil(totalRecords / limit);

        const tickets = await TicketModel.findAll({
            where: whereCondition,
            limit,
            offset: skip,
            include: [{ model: User }],
        });

        res.status(200).json({
            status: "success",
            results: tickets.length,
            tickets,
            totalRecords,
            totalPages,
            currentPage: page,
        });
    } catch (error) {
        res.status(500).json({
            status: "error",
            message: error.message,
        });
    }
});

router.get("/overdue", Auth, async (req, res) => {

    const user = await User.findByPk(req.user.id, { attributes: { exclude: ["password"] } });

    try {
        const page = parseInt(req.query.page) || 1;
        const limit = parseInt(req.query.limit) || 10;
        const skip = (page - 1) * limit;

        const filters = {
            ...(req.query.category && { category: req.query.category }),
            ...(req.query.system && { system: req.query.system }),
            ...(req.query.level && { level: req.query.level }),
            ...(req.query.facility && { facility: req.query.facility }),
            agentId: user.id,
            status: 'overdue'
        };

        const hasFilters = Object.keys(filters).length > 0;
        const whereCondition = hasFilters ? filters : {};

        const totalRecords = await TicketModel.count({ where: whereCondition });
        const totalPages = Math.ceil(totalRecords / limit);

        const tickets = await TicketModel.findAll({
            where: whereCondition,
            limit,
            offset: skip,
            include: [{ model: User }],
        });

        res.status(200).json({
            status: "success",
            results: tickets.length,
            tickets,
            totalRecords,
            totalPages,
            currentPage: page,
        });
    } catch (error) {
        res.status(500).json({
            status: "error",
            message: error.message,
        });
    }
});

router.get("/inprogress", Auth, async (req, res) => {

    const user = await User.findByPk(req.user.id, { attributes: { exclude: ["password"] } });

    try {
        const page = parseInt(req.query.page) || 1;
        const limit = parseInt(req.query.limit) || 10;
        const skip = (page - 1) * limit;

        const filters = {
            ...(req.query.category && { category: req.query.category }),
            ...(req.query.system && { system: req.query.system }),
            ...(req.query.level && { level: req.query.level }),
            ...(req.query.facility && { facility: req.query.facility }),
            agentId: user.id,
            status: 'inprogress'
        };

        const hasFilters = Object.keys(filters).length > 0;
        const whereCondition = hasFilters ? filters : {};

        const totalRecords = await TicketModel.count({ where: whereCondition });
        const totalPages = Math.ceil(totalRecords / limit);

        const tickets = await TicketModel.findAll({
            where: whereCondition,
            limit,
            offset: skip,
            include: [{ model: User }],
        });

        res.status(200).json({
            status: "success",
            results: tickets.length,
            tickets,
            totalRecords,
            totalPages,
            currentPage: page,
        });
    } catch (error) {
        res.status(500).json({
            status: "error",
            message: error.message,
        });
    }
});

router.get("/count", Auth, async (req, res) => {

    const user = await User.findByPk(req.user.id, { attributes: { exclude: ["password"] } });

    try {
        const tickets = await TicketModel.findAll({
            where: {
                agentId: user.id,
            },
        });

        let open = 0;
        let closed = 0;
        let inprogress = 0;
        let overdue = 0;

        tickets.forEach((ticket) => {
            switch (ticket.status) {
                case "inprogress":
                    inprogress++;
                    break;
                case "closed":
                    closed++;
                    break;
                case "open":
                    open++;
                    break;
                case "overdue":
                    overdue++;
                    break;
                default:
                    break;
            }
        });

        res.status(200).json({
            status: "success",
            results: tickets.length,
            tickets,
            statusCounts: {
                open: open,
                closed: closed,
                inprogress: inprogress,
                overdue: overdue,
            },
        });
    } catch (error) {
        res.status(500).json({
            status: "error",
            message: error.message,
        });
    }
});

export default router;