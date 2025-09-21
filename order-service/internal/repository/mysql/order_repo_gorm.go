package mysql

import (
	"errors"
	"log"
	"order-service/internal/domain"
	"order-service/internal/repository"

	"gorm.io/gorm"
)

type orderRepo struct {
    db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) repository.OrderRepository {
    return &orderRepo{db: db}
}

// CRITICAL FIX: Ensure ID is properly assigned and returned
func (r *orderRepo) Save(order *domain.Order) error {
    // Use Create which will populate the ID field
    result := r.db.Create(order)
    if result.Error != nil {
        log.Printf("Database save error: %v", result.Error)
        return result.Error
    }
    
    // Verify that ID was assigned
    if order.ID == 0 {
        log.Printf("WARNING: Order saved but ID is still 0. Rows affected: %d", result.RowsAffected)
        return errors.New("failed to assign order ID")
    }
    
    log.Printf("Order saved successfully with ID: %d", order.ID)
    return nil
}

// Batch save with proper error handling
func (r *orderRepo) SaveBatch(orders []*domain.Order) error {
    if len(orders) == 0 {
        return nil
    }
    
    // Use transaction for batch insert
    tx := r.db.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()
    
    // Create in batches
    for i := 0; i < len(orders); i += 100 { // Process in smaller chunks
        end := i + 100
        if end > len(orders) {
            end = len(orders)
        }
        
        batch := orders[i:end]
        result := tx.Create(&batch)
        if result.Error != nil {
            tx.Rollback()
            log.Printf("Batch save error: %v", result.Error)
            return result.Error
        }
        
        // Verify all orders in this batch got IDs
        for _, order := range batch {
            if order.ID == 0 {
                tx.Rollback()
                return errors.New("batch insert failed to assign IDs")
            }
        }
        
        log.Printf("Batch chunk %d-%d saved successfully", i, end)
    }
    
    err := tx.Commit().Error
    if err != nil {
        log.Printf("Batch commit error: %v", err)
        return err
    }
    
    log.Printf("Batch of %d orders saved successfully", len(orders))
    return nil
}

func (r *orderRepo) FindByID(id uint64) (*domain.Order, error) {
    var o domain.Order
    if err := r.db.First(&o, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        log.Printf("FindByID error: %v", err)
        return nil, err
    }
    return &o, nil
}

func (r *orderRepo) FindByProductId(productId uint64) ([]domain.Order, error) {
    var out []domain.Order
    if err := r.db.Where("product_id = ?", productId).Order("created_at DESC").Find(&out).Error; err != nil {
        log.Printf("FindByProductId error: %v", err)
        return nil, err
    }
    
    if len(out) == 0 {
        return nil, nil
    }
    return out, nil
}