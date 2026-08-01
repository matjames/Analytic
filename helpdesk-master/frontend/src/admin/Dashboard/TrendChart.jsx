import React, { useState, useEffect } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend } from 'recharts';

const TrendChart = ({ data }) => {
    const [issueData, setIssueData] = useState([]);

    useEffect(() => {
        const processedData = processData(data);
        setIssueData(processedData);
    }, [data]);

    const processData = (data) => {
        const issueMap = {};
        data && data.forEach((issue) => {
            const date = new Date(issue.reportedDate).toLocaleDateString();
            if (!issueMap[date]) {
                issueMap[date] = { created: 0, closed: 0 };
            }
            issueMap[date].created++;
            if (issue.status === 'Closed') {
                issueMap[date].closed++;
            }
        });

        const processedData = Object.keys(issueMap).map((date) => ({
            date,
            created: issueMap[date].created,
            closed: issueMap[date].closed,
        }));

        return processedData;
    };

    return (
        <LineChart width={1200} height={300} data={issueData} margin={{ top: 20, right: 30, left: 20, bottom: 10 }}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="date" />
            <YAxis />
            <Tooltip />
            <Legend />
            <Line type="monotone" dataKey="created" stroke="#8884d8" activeDot={{ r: 8 }} />
            <Line type="monotone" dataKey="closed" stroke="#82ca9d" />
        </LineChart>
    );
};

export default TrendChart;
