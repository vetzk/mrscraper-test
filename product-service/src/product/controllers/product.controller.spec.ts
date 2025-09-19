import { Test, TestingModule } from '@nestjs/testing';
import { ProductController } from './product.controller';
import { ProductService } from '../services/product.service';
import { ClientProxy } from '@nestjs/microservices';
import { CreateProductDto } from '../dtos/create-product.dto';
import { Logger } from '@nestjs/common';

describe('ProductController', () => {
  let controller: ProductController;
  let productService: jest.Mocked<ProductService>;
  let client: jest.Mocked<ClientProxy>;

  const mockProduct = {
    id: 1,
    name: 'Test Product',
    price: 100,
    qty: 10,
    createdAt: new Date(),
  };

  beforeEach(async () => {
    const mockProductService = {
      findOne: jest.fn(),
      create: jest.fn(),
      decrementQty: jest.fn(),
    };

    const mockClient = {
      emit: jest.fn().mockReturnValue({
        toPromise: jest.fn().mockResolvedValue(undefined),
      }),
      send: jest.fn(),
      connect: jest.fn(),
      close: jest.fn(),
    };

    const module: TestingModule = await Test.createTestingModule({
      controllers: [ProductController],
      providers: [
        {
          provide: ProductService,
          useValue: mockProductService,
        },
        {
          provide: 'PRODUCT_PUBLISHER',
          useValue: mockClient,
        },
      ],
    }).compile();

    controller = module.get<ProductController>(ProductController);
    productService = module.get(ProductService);
    client = module.get('PRODUCT_PUBLISHER');

    // Mock Logger to avoid console output during tests
    jest.spyOn(Logger.prototype, 'log').mockImplementation();
    jest.spyOn(Logger.prototype, 'warn').mockImplementation();
  });

  afterEach(() => {
    jest.clearAllMocks();
    jest.restoreAllMocks();
  });

  it('should be defined', () => {
    expect(controller).toBeDefined();
  });

  describe('findOne', () => {
    it('should return a product', async () => {
      // Arrange
      const productId = 1;
      productService.findOne.mockResolvedValue(mockProduct);

      // Act
      const result = await controller.findOne(productId);

      // Assert
      expect(result).toEqual(mockProduct);
      expect(productService.findOne).toHaveBeenCalledWith(productId);
    });
  });

  describe('create', () => {
    it('should create a new product', async () => {
      // Arrange
      const createProductDto: CreateProductDto = {
        name: 'New Product',
        price: 200,
        qty: 5,
      };
      const createdProduct = { ...mockProduct, ...createProductDto };
      productService.create.mockResolvedValue(createdProduct);

      // Act
      const result = await controller.create(createProductDto);

      // Assert
      expect(result).toEqual(createdProduct);
      expect(productService.create).toHaveBeenCalledWith(createProductDto);
    });
  });

  describe('handleOrderCreated', () => {
    const orderData = {
      orderId: 123,
      productId: 1,
      totalPrice: 1000,
      createdAt: new Date(),
    };

    it('should handle order created and confirm quantity when product available', async () => {
      // Arrange
      const updatedProduct = { ...mockProduct, qty: 9 };
      productService.decrementQty.mockResolvedValue(updatedProduct);

      // Act
      await controller.handleOrderCreated(orderData);

      // Assert
      expect(productService.decrementQty).toHaveBeenCalledWith(
        orderData.productId,
      );
      expect(client.emit).toHaveBeenCalledWith('order.qty_confirmed', {
        orderId: orderData.orderId,
      });
      expect(Logger.prototype.log).toHaveBeenCalledWith(
        `Received order.created for order ${orderData.orderId}, product ${orderData.productId}`,
      );
    });

    it('should emit qty_failed when product not found or unavailable', async () => {
      // Arrange
      productService.decrementQty.mockResolvedValue(null);

      // Act
      await controller.handleOrderCreated(orderData);

      // Assert
      expect(productService.decrementQty).toHaveBeenCalledWith(
        orderData.productId,
      );
      expect(client.emit).toHaveBeenCalledWith('order.qty_failed', {
        orderId: orderData.orderId,
        reason: 'product_not_found_or_unavailable',
      });
      expect(Logger.prototype.log).toHaveBeenCalledWith(
        `Received order.created for order ${orderData.orderId}, product ${orderData.productId}`,
      );
    });
  });

  describe('handleAny', () => {
    it('should log wildcard pattern warnings', async () => {
      // Arrange
      const mockContext = {
        getPattern: jest.fn().mockReturnValue('some.unknown.pattern'),
      };
      const testData = { some: 'data' };

      // Act
      await controller.handleAny(testData, mockContext as any);

      // Assert
      expect(mockContext.getPattern).toHaveBeenCalled();
      expect(Logger.prototype.warn).toHaveBeenCalledWith(
        '⚠️ Wildcard caught pattern: some.unknown.pattern',
      );
      expect(Logger.prototype.warn).toHaveBeenCalledWith(
        `Raw data: ${JSON.stringify(testData)}`,
      );
    });
  });
});
