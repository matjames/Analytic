import React, { useState, useEffect } from 'react';
import { PieChart, Pie, Cell, Tooltip, Legend } from 'recharts';

const IssueDistribution = ({ data, indicator }) => {
    const [distributionData, setDistributionData] = useState([]);

    useEffect(() => {
        // Process data to count issues based on the indicator
        const processedData = processData(data, indicator);
        setDistributionData(processedData);
    }, [data, indicator]);

    const processData = (data, indicator) => {
        const distributionMap = {};
        data.forEach((issue) => {
            const indicatorValue = issue[indicator];
            if (!distributionMap[indicatorValue]) {
                distributionMap[indicatorValue] = 0;
            }
            distributionMap[indicatorValue]++;
        });

        // Convert distributionMap to an array of objects
        const processedData = Object.keys(distributionMap).map((value) => ({
            name: value,
            value: distributionMap[value],
        }));

        return processedData;
    };

    const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#AF19FF', '#8884d8', "#82ca9d", "#ffb61d", "#43b8f9"];

    return (
        <PieChart width={400} height={280}>
            <Pie
                data={distributionData}
                cx={165}
                cy={120}
                labelLine={false}
                label={({
                    cx,
                    cy,
                    midAngle,
                    innerRadius,
                    outerRadius,
                    value,
                    index
                }) => {
                    const RADIAN = Math.PI / 180;
                    const radius = 25 + innerRadius + (outerRadius - innerRadius);
                    const x = cx + radius * Math.cos(-midAngle * RADIAN);
                    const y = cy + radius * Math.sin(-midAngle * RADIAN);
                    return (
                        <text
                            x={x}
                            y={y}
                            fill="#8884d8"
                            textAnchor={x > cx ? 'start' : 'end'}
                            dominantBaseline="central"
                        >
                            {distributionData[index].name} ({value})
                        </text>
                    );
                }}
                outerRadius={80}
                fill="#8884d8"
                dataKey="value"
                nameKey="name"
            >
                {distributionData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                ))}
            </Pie>
            <Tooltip />
            <Legend />
        </PieChart>
    );
};

export default IssueDistribution;