-- 006_create_dice_results_table.up.sql
CREATE TABLE IF NOT EXISTS dice_results (
    expression TEXT PRIMARY KEY,
    data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_dice_results_expression ON dice_results(expression);

-- Create a function to update the updated_at column
CREATE OR REPLACE FUNCTION update_dice_results_updated_at()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create a trigger to automatically update the updated_at column
CREATE TRIGGER update_dice_results_updated_at
    BEFORE UPDATE ON dice_results
    FOR EACH ROW
EXECUTE FUNCTION update_dice_results_updated_at();