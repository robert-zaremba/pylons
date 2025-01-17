import 'dart:io';
import 'package:dartz/dartz.dart';
import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:get_it/get_it.dart';
import 'package:mockito/mockito.dart';
import 'package:provider/provider.dart';
import 'package:pylons_wallet/generated/locale_keys.g.dart';
import 'package:pylons_wallet/pages/purchase_item/purchase_item_screen.dart';
import 'package:pylons_wallet/pages/purchase_item/purchase_item_view_model.dart';
import 'package:pylons_wallet/pages/purchase_item/widgets/buy_nft_button.dart';
import 'package:pylons_wallet/stores/wallet_store.dart';
import 'package:pylons_wallet/utils/constants.dart';
import '../../../mocks/mock_constants.dart';
import '../../../mocks/mock_wallet_public_info.dart';
import '../../../mocks/mock_wallet_store.dart';
import '../../../mocks/purchase_item_view_model.mocks.dart';
import '../../extension/size_extension.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  final WalletsStore walletStore = MockWalletStore();
  final PurchaseItemViewModel viewModel = MockPurchaseItemViewModel();
  GetIt.I.registerLazySingleton<PurchaseItemViewModel>(() => viewModel);

  testWidgets("check buy button is in expanded view or not", (tester) async {
    GetIt.I.registerLazySingleton<WalletsStore>(() => walletStore);

    when(viewModel.collapsed).thenAnswer((realInvocation) => false);
    when(viewModel.nft).thenAnswer((realInvocation) => MOCK_NFT_FREE_VIDEO);
    when(viewModel.showBuyNowButton(isPlatformAndroid: Platform.isAndroid)).thenAnswer((realInvocation) => true);
    await tester.testAppForWidgetTesting(
      ChangeNotifierProvider<PurchaseItemViewModel>.value(
        value: viewModel,
        child: PurchaseItemScreen(
          nft: viewModel.nft,
        ),
      ),
    );
    await tester.pump();
    final expandedBuyButtonFinder = find.byKey(const Key(kExpandedBuyButtonKeyValue));
    expect(expandedBuyButtonFinder, findsOneWidget);
  });

  testWidgets('should show the Free Drop NFT Buy Button and make sure user is able to tap', (tester) async {
    final buyButtonFinder = find.text(LocaleKeys.claim_now.tr());
    var isTapped = false;

    await tester.setScreenSize();
    await tester.testAppForWidgetTesting(ChangeNotifierProvider.value(
        value: viewModel,
        builder: (context, child) {
          return Material(
            child: BuyNFTButton(
              onTapped: () {
                isTapped = true;
              },
              nft: MOCK_NFT_FREE_IMAGE,
            ),
          );
        }));

    expect(buyButtonFinder, findsOneWidget);
    await tester.tap(buyButtonFinder);

    expect(true, isTapped);
  });

  testWidgets('should show the NFT Buy Button and make sure user is able to tap', (tester) async {
    int counter = 0;
    final buyButtonFinder = find.text(LocaleKeys.buy_now.tr());
    await tester.setScreenSize();
    await tester.testAppForWidgetTesting(ChangeNotifierProvider.value(
      value: viewModel,
      child: Material(
        child: BuyNFTButton(
          key: const Key(MOCK_BUY_BUTTON_KEY_VALUE),
          onTapped: () {
            counter++;
          },
          nft: MOCK_NFT_PREMIUM,
        ),
      ),
    ));
    final buyButton = find.byKey(const Key(MOCK_BUY_BUTTON_KEY_VALUE));
    expect(counter, 0);
    await tester.tap(buyButton);
    await tester.pump();
    expect(buyButtonFinder, findsOneWidget);
    expect(counter, 1);
  });

  testWidgets('should show remaining quantity & total quantity in Free NFT Buy Button ', (tester) async {
    await tester.setScreenSize();
    await tester.testAppForWidgetTesting(ChangeNotifierProvider.value(
      value: viewModel,
      child: Material(
        child: BuyNFTButton(
          key: const Key(MOCK_BUY_BUTTON_KEY_VALUE),
          onTapped: () {},
          nft: MOCK_NFT_FREE_IMAGE,
        ),
      ),
    ));
    final buyButton = find.text(LocaleKeys.remaining_and_total_editions.tr(
      namedArgs: {
        'remaining': '${MOCK_NFT_FREE_IMAGE.quantity - MOCK_NFT_FREE_IMAGE.amountMinted}',
        'total': '${MOCK_NFT_FREE_IMAGE.quantity}',
      },
    ));
    expect(buyButton, findsOneWidget);
  });

  testWidgets('should show claim now in Free NFT Buy Button ', (tester) async {
    await tester.setScreenSize();
    await tester.testAppForWidgetTesting(ChangeNotifierProvider.value(
      value: viewModel,
      child: Material(
        child: BuyNFTButton(
          key: const Key(MOCK_BUY_BUTTON_KEY_VALUE),
          onTapped: () {},
          nft: MOCK_NFT_FREE_IMAGE,
        ),
      ),
    ));
    final buyButton = find.text(LocaleKeys.claim_now.tr());
    expect(buyButton, findsOneWidget);
  });

  testWidgets('should show buy now in Premium NFT Buy Button ', (tester) async {
    await tester.setScreenSize();
    await tester.testAppForWidgetTesting(ChangeNotifierProvider.value(
      value: viewModel,
      child: Material(
        child: BuyNFTButton(
          key: const Key(MOCK_BUY_BUTTON_KEY_VALUE),
          onTapped: () {},
          nft: MOCK_NFT_PREMIUM,
        ),
      ),
    ));
    final buyButton = find.text(LocaleKeys.buy_now.tr());
    expect(buyButton, findsOneWidget);
  });

  testWidgets("Checkout Dialog should show on Buy Now Button Click", (tester) async {
    when(viewModel.collapsed).thenAnswer((realInvocation) => false);
    when(viewModel.nft).thenAnswer((realInvocation) => MOCK_NFT_PREMIUM);
    when(viewModel.accountPublicInfo).thenAnswer((realInvocation) => MockAccountPublicInfo());
    when(viewModel.showBuyNowButton(isPlatformAndroid: Platform.isAndroid)).thenAnswer((realInvocation) => true);
    when(viewModel.shouldShowSwipeToBuy(
      selectedDenom: MOCK_NFT_PREMIUM.denom,
      requiredAmount: double.parse(MOCK_NFT_PREMIUM.price) / kBigIntBase,
    )).thenAnswer((realInvocation) => Future.value(const Right(true)));
    await tester.setScreenSize();
    await tester.testAppForWidgetTesting(
      ChangeNotifierProvider<PurchaseItemViewModel>.value(
        value: viewModel,
        child: PurchaseItemScreen(
          nft: viewModel.nft,
        ),
      ),
    );
    await tester.pump();
    final buyNFTButton = find.byKey(const Key(kExpandedBuyButtonKeyValue));
    expect(buyNFTButton, findsOneWidget);
    await tester.tap(buyNFTButton);
    await tester.pump();
    final checkoutDialog = find.byKey(const Key(kCheckoutDialogKey));
    expect(checkoutDialog, findsOneWidget);
  });
}
