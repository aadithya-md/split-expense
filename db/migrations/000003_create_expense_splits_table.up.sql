CREATE TABLE expense_splits (
    id INT AUTO_INCREMENT PRIMARY KEY,
    expense_id INT NOT NULL,
    user_id INT NOT NULL,
    amount_paid DECIMAL(10, 2) NOT NULL,
    amount_owed DECIMAL(10, 2) NOT NULL,
    FOREIGN KEY (expense_id) REFERENCES expenses(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    INDEX idx_expense_splits_user_id (user_id)
);
