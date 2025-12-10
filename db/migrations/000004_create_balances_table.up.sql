CREATE TABLE balances (
    user1_id INT NOT NULL,
    user2_id INT NOT NULL,
    balance DECIMAL(10, 2) NOT NULL,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (user1_id, user2_id),
    FOREIGN KEY (user1_id) REFERENCES users(id),
    FOREIGN KEY (user2_id) REFERENCES users(id),
    INDEX idx_balances_user2_id (user2_id)
);
