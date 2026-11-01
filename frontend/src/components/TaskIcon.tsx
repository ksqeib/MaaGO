import React from "react";
import {
  FaCrosshairs,
  FaGift,
  FaIndustry,
  FaRocket,
  FaShoppingCart,
  FaUsers,
} from "react-icons/fa";

import type { TaskIcon as TaskIconName } from "@/data/tasks";

const icons: Record<TaskIconName, React.ReactNode> = {
  rocket: <FaRocket />,
  swords: <FaCrosshairs />,
  factory: <FaIndustry />,
  users: <FaUsers />,
  shopping: <FaShoppingCart />,
  gift: <FaGift />,
};

export const TaskIcon = ({
  name,
  size = 16,
}: {
  name: TaskIconName;
  size?: number;
}) => (
  <span style={{ fontSize: size, lineHeight: 1 }}>{icons[name]}</span>
);
