import { Test, TestingModule } from '@nestjs/testing';
import { getRepositoryToken } from '@nestjs/typeorm';
import { CACHE_MANAGER } from '@nestjs/cache-manager';
import { NotFoundException } from '@nestjs/common';
import { Repository } from 'typeorm';
import { Cache } from 'cache-manager';
import { ClientProxy } from '@nestjs/microservices';
import { ProductService } from './product.service';
import { Product } from '../domain/product';

describe('ProductService', () => {
  let service: ProductService;
  let productRepo: jest.Mocked<Repository<Product>>;
  let cacheManager: jest.Mocked<Cache>;
  let client: jest.Mocked<ClientProxy>;

  const mockProduct: Product = {
    id: 1,
    name: 'Test Product',
    price: 100,
    qty: 10,
    createdAt: new Date(),
  };

  beforeEach(async () => {
    const mockProductRepo = {
      findOne: jest.fn(),
      create: jest.fn(),
      save: jest.fn(),
    };

    const mockCacheManager = {
      get: jest.fn(),
      set: jest.fn(),
    };

    const mockClient = {
      emit: jest.fn(),
    };

    const module: TestingModule = await Test.createTestingModule({
      providers: [
        ProductService,
        {
          provide: getRepositoryToken(Product),
          useValue: mockProductRepo,
        },
        {
          provide: CACHE_MANAGER,
          useValue: mockCacheManager,
        },
        {
          provide: 'PRODUCT_PUBLISHER',
          useValue: mockClient,
        },
      ],
    }).compile();

    service = module.get<ProductService>(ProductService);
    productRepo = module.get(getRepositoryToken(Product));
    cacheManager = module.get(CACHE_MANAGER);
    client = module.get('PRODUCT_PUBLISHER');
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('findOne', () => {
    it('should return cached product when cache hit', async () => {
      // Arrange
      const productId = 1;
      cacheManager.get.mockResolvedValue(mockProduct);

      // Act
      const result = await service.findOne(productId);

      // Assert
      expect(result).toEqual(mockProduct);
      expect(cacheManager.get).toHaveBeenCalledWith(`product:${productId}`);
      expect(productRepo.findOne).not.toHaveBeenCalled();
      expect(cacheManager.set).not.toHaveBeenCalled();
    });

    it('should fetch from database and cache when cache miss', async () => {
      // Arrange
      const productId = 1;
      cacheManager.get.mockResolvedValue(null);
      productRepo.findOne.mockResolvedValue(mockProduct);

      // Act
      const result = await service.findOne(productId);

      // Assert
      expect(result).toEqual(mockProduct);
      expect(cacheManager.get).toHaveBeenCalledWith(`product:${productId}`);
      expect(productRepo.findOne).toHaveBeenCalledWith({
        where: { id: productId },
      });
      expect(cacheManager.set).toHaveBeenCalledWith(
        `product:${productId}`,
        mockProduct,
        60_000,
      );
    });

    it('should throw NotFoundException when product not found', async () => {
      // Arrange
      const productId = 999;
      cacheManager.get.mockResolvedValue(null);
      productRepo.findOne.mockResolvedValue(null);

      // Act & Assert
      await expect(service.findOne(productId)).rejects.toThrow(
        new NotFoundException(`Product ${productId} not found`),
      );
    });
  });

  describe('create', () => {
    it('should create and save product, then emit event', async () => {
      // Arrange
      const productData = {
        name: 'New Product',
        price: 200,
        qty: 5,
        description: 'New Description',
      };
      const createdProduct = { ...mockProduct, ...productData };

      productRepo.create.mockReturnValue(createdProduct as Product);
      productRepo.save.mockResolvedValue(createdProduct);

      // Act
      const result = await service.create(productData);

      // Assert
      expect(productRepo.create).toHaveBeenCalledWith(productData);
      expect(productRepo.save).toHaveBeenCalledWith(createdProduct);
      expect(client.emit).toHaveBeenCalledWith(
        'product_created',
        createdProduct,
      );
      expect(result).toEqual(createdProduct);
    });
  });

  describe('decrementQty', () => {
    it('should decrement quantity when product exists and has stock', async () => {
      // Arrange
      const productId = 1;
      const productWithStock = { ...mockProduct, qty: 5 };
      const updatedProduct = { ...productWithStock, qty: 4 };

      productRepo.findOne.mockResolvedValue(productWithStock);
      productRepo.save.mockResolvedValue(updatedProduct);

      // Act
      const result = await service.decrementQty(productId);

      // Assert
      expect(productRepo.findOne).toHaveBeenCalledWith({
        where: { id: productId },
      });
      expect(productRepo.save).toHaveBeenCalledWith(updatedProduct);
      expect(result).toEqual(updatedProduct);
      expect(result?.qty).toBe(4);
    });

    it('should return null when product not found', async () => {
      // Arrange
      const productId = 999;
      productRepo.findOne.mockResolvedValue(null);

      // Act
      const result = await service.decrementQty(productId);

      // Assert
      expect(result).toBeNull();
      expect(productRepo.save).not.toHaveBeenCalled();
    });

    it('should return null when product quantity is zero', async () => {
      // Arrange
      const productId = 1;
      const outOfStockProduct = { ...mockProduct, qty: 0 };
      productRepo.findOne.mockResolvedValue(outOfStockProduct);

      // Act
      const result = await service.decrementQty(productId);

      // Assert
      expect(result).toBeNull();
      expect(productRepo.save).not.toHaveBeenCalled();
    });
  });
});
